package intent

import (
	powerai "orgine.com/ai-team/power-ai-framework-v4"
	"orgine.com/ai-team/power-ai-framework-v4/middleware/server"
)

type Classifier interface {
	Classify(app *powerai.AgentApp, req *server.AgentRequest) (Result, error)
}
