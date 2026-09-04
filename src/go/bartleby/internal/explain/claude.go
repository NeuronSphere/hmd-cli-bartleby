package explain

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// DefaultModel is used unless BARTLEBY_EXPLAIN_MODEL or --model says otherwise.
// A build log is a reasoning problem — the cause is often two inferences away
// from anything the log states — so the default is the strongest model rather
// than the cheapest.
const DefaultModel = "claude-opus-5"

// DefaultMaxTokens caps the explanation. It is generous: a good answer includes a
// corrected snippet.
const DefaultMaxTokens = 8192

// ErrNoCredentials reports that no Anthropic credentials could be found.
var ErrNoCredentials = errors.New("no Anthropic credentials")

// Requester turns a payload into an explanation. The interface exists so the
// command can be tested without a network or a key.
type Requester interface {
	Explain(ctx context.Context, system, user string) (string, error)
}

// Claude calls the Messages API once.
type Claude struct {
	// Model defaults to DefaultModel.
	Model string
	// MaxTokens defaults to DefaultMaxTokens.
	MaxTokens int64
	// APIKey is optional: when empty the SDK resolves credentials itself, which
	// covers ANTHROPIC_API_KEY, ANTHROPIC_AUTH_TOKEN, an `ant auth login`
	// profile, and workload identity federation.
	APIKey string
	// Stream, when set, receives the answer as it arrives so the user is not
	// staring at nothing.
	Stream io.Writer
}

// Explain sends one request and returns the answer.
//
// The response is streamed: with thinking on by default for the Opus family, a
// non-streaming request can sit silent for a long time, and a CLI that prints
// nothing looks broken.
func (c Claude) Explain(ctx context.Context, system, user string) (string, error) {
	model := c.Model
	if model == "" {
		model = DefaultModel
	}
	maxTokens := c.MaxTokens
	if maxTokens == 0 {
		maxTokens = DefaultMaxTokens
	}

	var opts []option.RequestOption
	if c.APIKey != "" {
		opts = append(opts, option.WithAPIKey(c.APIKey))
	}
	client := anthropic.NewClient(opts...)

	stream := client.Messages.NewStreaming(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(model),
		MaxTokens: maxTokens,
		System: []anthropic.TextBlockParam{{
			Text: system,
		}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(user)),
		},
	})

	message := anthropic.Message{}
	for stream.Next() {
		event := stream.Current()
		if err := message.Accumulate(event); err != nil {
			return "", fmt.Errorf("reading the response: %w", err)
		}

		if c.Stream == nil {
			continue
		}
		if delta, ok := event.AsAny().(anthropic.ContentBlockDeltaEvent); ok {
			if text, ok := delta.Delta.AsAny().(anthropic.TextDelta); ok {
				fmt.Fprint(c.Stream, text.Text)
			}
		}
	}

	if err := stream.Err(); err != nil {
		return "", fmt.Errorf("asking Claude: %w", err)
	}

	if message.StopReason == anthropic.StopReasonRefusal {
		return "", fmt.Errorf("Claude declined to answer (%s)", message.StopDetails.Category)
	}

	var b strings.Builder
	for _, block := range message.Content {
		if text, ok := block.AsAny().(anthropic.TextBlock); ok {
			b.WriteString(text.Text)
		}
	}

	answer := strings.TrimSpace(b.String())
	if answer == "" {
		return "", errors.New("Claude returned no text")
	}
	return answer, nil
}

// HasCredentials reports whether anything is available to authenticate with.
//
// This is a best-effort check used to fail early with a useful message. The SDK
// resolves more sources than an API key — an `ant auth login` profile, for one —
// so a false here is a reason to explain the options, not to refuse outright.
func HasCredentials(env func(string) string, profileExists func() bool) bool {
	if env == nil {
		return false
	}
	for _, key := range []string{"ANTHROPIC_API_KEY", "ANTHROPIC_AUTH_TOKEN"} {
		if strings.TrimSpace(env(key)) != "" {
			return true
		}
	}
	if env("ANTHROPIC_IDENTITY_TOKEN") != "" || env("ANTHROPIC_IDENTITY_TOKEN_FILE") != "" {
		return true
	}
	return profileExists != nil && profileExists()
}
