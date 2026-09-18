package observability

import (
	"encoding/json"
	"log"
	"time"
)

func Log(level, msg string, kv map[string]any) {
	payload := map[string]any{
		"ts":    time.Now().UTC().Format(time.RFC3339Nano),
		"level": level,
		"msg":   msg,
	}
	for k, v := range kv {
		payload[k] = v
	}
	b, err := json.Marshal(payload)
	if err != nil {
		log.Printf("{\"level\":\"error\",\"msg\":\"log marshal failed\",\"err\":%q}", err.Error())
		return
	}
	log.Print(string(b))
}
