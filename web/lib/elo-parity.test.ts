import { describe, expect, it } from 'vitest';
import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, resolve } from 'node:path';
import { calculateDuelEloDeltas } from './elo';

// Cross-implementation parity: the committed golden file in tests/shared is
// produced by the production Go rating implementation
// (backend/internal/rating/rating_parity_test.go). The TypeScript
// implementation must reproduce its deltas and RD outputs exactly, so the two
// production implementations cannot silently diverge. The rating math is never
// reimplemented here.

const sharedDir = resolve(dirname(fileURLToPath(import.meta.url)), '../../tests/shared');

type ParityCase = {
  id: string;
  selfMMR: number;
  oppMMR: number;
  selfRD: number;
  oppRD: number;
  winner: 'self' | 'opp' | 'draw';
  selfIdleDays: number | null;
  oppIdleDays: number | null;
};

type ParityResult = {
  id: string;
  selfDelta: number;
  oppDelta: number;
  selfRD: number;
  oppRD: number;
};

const inputs = JSON.parse(readFileSync(resolve(sharedDir, 'rating-cases.json'), 'utf8')) as {
  fixedNow: string;
  cases: ParityCase[];
};
const golden = JSON.parse(readFileSync(resolve(sharedDir, 'rating-expected.json'), 'utf8')) as {
  results: ParityResult[];
};

const fixedNow = new Date(inputs.fixedNow);

function updatedAt(idleDays: number | null): Date | undefined {
  if (idleDays === null) return undefined;
  return new Date(fixedNow.getTime() - idleDays * 86_400_000);
}

describe('TypeScript rating parity with the Go implementation', () => {
  it('covers a non-empty shared corpus', () => {
    expect(inputs.cases.length).toBeGreaterThan(0);
    expect(golden.results).toHaveLength(inputs.cases.length);
  });

  for (let i = 0; i < inputs.cases.length; i++) {
    const c = inputs.cases[i];
    const want = golden.results[i];
    it(`case ${c.id}`, () => {
      const got = calculateDuelEloDeltas(
        c.selfMMR,
        c.oppMMR,
        c.winner,
        c.selfRD,
        c.oppRD,
        updatedAt(c.selfIdleDays),
        updatedAt(c.oppIdleDays),
        fixedNow,
      );
      expect(got.selfDelta).toBe(want.selfDelta);
      expect(got.opponentDelta).toBe(want.oppDelta);
      expect(got.selfRatingRd).toBeCloseTo(want.selfRD, 9);
      expect(got.opponentRatingRd).toBeCloseTo(want.oppRD, 9);
    });
  }
});
