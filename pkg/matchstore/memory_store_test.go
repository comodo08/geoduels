package matchstore

import (
	"math"
	"sort"
	"sync"
	"time"

	"geoduels/pkg/contracts"
)

type memoryStore struct {
	mu      sync.Mutex
	queues  map[QueuePool][]ticket
	matches map[QueuePool]map[string]contracts.MatchFound
}

func newMemory() Store {
	return &memoryStore{
		queues: map[QueuePool][]ticket{
			QueuePoolRegistered: {},
		},
		matches: map[QueuePool]map[string]contracts.MatchFound{
			QueuePoolRegistered: {},
		},
	}
}

func (m *memoryStore) Join(pool QueuePool, ruleset contracts.GameRuleset, req contracts.QueueJoinRequest) (contracts.QueueJoinResponse, *contracts.MatchFound, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	queue := m.queues[pool]
	for _, t := range queue {
		if t.UserID == req.UserID {
			return contracts.QueueJoinResponse{TicketID: t.ID, Status: "queued"}, nil, nil
		}
	}
	for other := range m.queues {
		if other == pool {
			continue
		}
		m.leaveLocked(other, req.UserID)
	}
	name := req.DisplayName
	if name == "" {
		name = req.UserID
	}
	t := ticket{
		ID:                ticketID(req.UserID),
		UserID:            req.UserID,
		DisplayName:       name,
		AvatarURL:         req.AvatarURL,
		MMR:               req.MMR,
		RatingRD:          req.RatingRD,
		SeasonID:          req.SeasonID,
		RankedGamesPlayed: req.RankedGamesPlayed,
		IsGuest:           req.IsGuest,
		Ruleset:           contracts.NormalizeRuleset(ruleset),
		JoinedAtUnixMS:    time.Now().UnixMilli(),
	}
	queue = append(queue, t)
	sort.Slice(queue, func(i, j int) bool { return queue[i].JoinedAtUnixMS < queue[j].JoinedAtUnixMS })
	m.queues[pool] = queue
	return contracts.QueueJoinResponse{TicketID: t.ID, Status: "queued"}, nil, nil
}

func (m *memoryStore) Leave(pool QueuePool, rulesets []contracts.GameRuleset, userID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.leaveLocked(pool, userID)
	return nil
}

func (m *memoryStore) LeaveAllRulesets(pool QueuePool, userID string) error {
	return m.Leave(pool, nil, userID)
}

func (m *memoryStore) leaveLocked(pool QueuePool, userID string) {
	queue := m.queues[pool]
	out := queue[:0]
	for _, t := range queue {
		if t.UserID != userID {
			out = append(out, t)
		}
	}
	m.queues[pool] = out
	delete(m.matches[pool], userID)
}

func (m *memoryStore) Heartbeat(pool QueuePool, rulesets []contracts.GameRuleset, userID string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.matches[pool][userID]; ok {
		return QueuePresenceMatched, nil
	}
	for _, t := range m.queues[pool] {
		if t.UserID == userID {
			return QueuePresenceQueueing, nil
		}
	}
	return QueuePresenceMissing, nil
}

func (m *memoryStore) Poll(pool QueuePool, rulesets []contracts.GameRuleset, userID string) (*contracts.MatchFound, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	mf, ok := m.matches[pool][userID]
	if !ok {
		return nil, nil
	}
	delete(m.matches[pool], userID)
	return &mf, nil
}

func (m *memoryStore) IsQueued(pool QueuePool, rulesets []contracts.GameRuleset, userID string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, t := range m.queues[pool] {
		if t.UserID == userID {
			return true, nil
		}
	}
	return false, nil
}

func (m *memoryStore) RunMatchmaking(pool QueuePool, ruleset contracts.GameRuleset, limit int) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if limit <= 0 {
		limit = 50
	}
	matched := 0
	for {
		if matched >= limit {
			break
		}
		matchedThisPass := false
		queue := append([]ticket(nil), m.queues[pool]...)
		nowMS := time.Now().UnixMilli()
		for _, t := range queue {
			if matched >= limit {
				break
			}
			if nowMS-t.JoinedAtUnixMS < mutualMatchWaitMS {
				continue
			}
			if m.tryMatchLocked(pool, t.UserID, nowMS) {
				matched++
				matchedThisPass = true
			}
		}
		if !matchedThisPass {
			break
		}
	}
	return matched, nil
}

func (m *memoryStore) tryMatchLocked(pool QueuePool, userID string, nowMS int64) bool {
	queue := m.queues[pool]
	selfIdx := -1
	for i, t := range queue {
		if t.UserID == userID {
			selfIdx = i
			break
		}
	}
	if selfIdx < 0 {
		return false
	}
	self := queue[selfIdx]
	remaining := append([]ticket{}, queue[:selfIdx]...)
	remaining = append(remaining, queue[selfIdx+1:]...)
	bestIdx := bestMatchIndex(remaining, self, nowMS)
	if bestIdx < 0 {
		return false
	}
	op := remaining[bestIdx]
	remaining = append(remaining[:bestIdx], remaining[bestIdx+1:]...)
	m.queues[pool] = remaining
	match := matchFromTickets(op, self)
	m.matches[pool][op.UserID] = match
	m.matches[pool][self.UserID] = match
	return true
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func allowedMatchDiff(nowMS, joinedA, joinedB int64) int {
	waitMS := nowMS - joinedA
	if other := nowMS - joinedB; other > waitMS {
		waitMS = other
	}
	if waitMS < 0 {
		waitMS = 0
	}
	allowed := baseMatchWindowMMR
	if matchExpandEveryMS > 0 {
		allowed += int(waitMS/matchExpandEveryMS) * matchExpandStepMMR
	}
	if allowed > maxMatchWindowMMR {
		return maxMatchWindowMMR
	}
	return allowed
}

func bestMatchIndex(queue []ticket, self ticket, nowMS int64) int {
	bestIdx := -1
	bestDiff := math.MaxInt
	for i, q := range queue {
		if q.UserID == self.UserID {
			continue
		}
		d := abs(q.MMR - self.MMR)
		if waitedLongEnough(nowMS, self.JoinedAtUnixMS, q.JoinedAtUnixMS) && d <= allowedMatchDiff(nowMS, self.JoinedAtUnixMS, q.JoinedAtUnixMS) && d < bestDiff {
			bestDiff = d
			bestIdx = i
		}
	}
	return bestIdx
}

func waitedLongEnough(nowMS, joinedA, joinedB int64) bool {
	return nowMS-joinedA >= mutualMatchWaitMS && nowMS-joinedB >= mutualMatchWaitMS
}
