package service

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

type bufferedResponsesAccumulator struct {
	response *apicompat.ResponsesResponse
}

func newBufferedResponsesAccumulator(fallbackModel, requestID string) *bufferedResponsesAccumulator {
	respID := strings.TrimSpace(requestID)
	if respID == "" {
		respID = "resp_partial"
	}
	return &bufferedResponsesAccumulator{
		response: &apicompat.ResponsesResponse{
			ID:     respID,
			Object: "response",
			Model:  strings.TrimSpace(fallbackModel),
			Status: "incomplete",
		},
	}
}

func (a *bufferedResponsesAccumulator) applyEvent(evt *apicompat.ResponsesStreamEvent) {
	if a == nil || evt == nil {
		return
	}
	if evt.Response != nil {
		if strings.TrimSpace(evt.Response.ID) != "" {
			a.response.ID = strings.TrimSpace(evt.Response.ID)
		}
		if strings.TrimSpace(evt.Response.Model) != "" {
			a.response.Model = strings.TrimSpace(evt.Response.Model)
		}
		if evt.Response.Usage != nil {
			usage := *evt.Response.Usage
			a.response.Usage = &usage
		}
	}

	switch evt.Type {
	case "response.output_item.added":
		if evt.Item == nil {
			return
		}
		item := a.ensureOutput(evt.OutputIndex, evt.Item.Type)
		*item = *evt.Item
		if item.Type == "message" {
			if item.Role == "" {
				item.Role = "assistant"
			}
			if item.Status == "" {
				item.Status = "incomplete"
			}
		}
	case "response.output_text.delta":
		if evt.Delta == "" {
			return
		}
		item := a.ensureOutput(evt.OutputIndex, "message")
		if item.Role == "" {
			item.Role = "assistant"
		}
		if item.Status == "" {
			item.Status = "incomplete"
		}
		part := ensureContentPart(item, evt.ContentIndex)
		if part.Type == "" {
			part.Type = "output_text"
		}
		part.Text += evt.Delta
	case "response.reasoning_summary_text.delta":
		if evt.Delta == "" {
			return
		}
		item := a.ensureOutput(evt.OutputIndex, "reasoning")
		summary := ensureSummaryPart(item, evt.SummaryIndex)
		if summary.Type == "" {
			summary.Type = "summary_text"
		}
		summary.Text += evt.Delta
	case "response.function_call_arguments.delta":
		if evt.Delta == "" {
			return
		}
		item := a.ensureOutput(evt.OutputIndex, "function_call")
		if strings.TrimSpace(evt.CallID) != "" && item.CallID == "" {
			item.CallID = strings.TrimSpace(evt.CallID)
		}
		if strings.TrimSpace(evt.Name) != "" && item.Name == "" {
			item.Name = strings.TrimSpace(evt.Name)
		}
		item.Arguments += evt.Delta
	}
}

func (a *bufferedResponsesAccumulator) hasUsefulOutput() bool {
	if a == nil || a.response == nil {
		return false
	}
	for _, item := range a.response.Output {
		switch item.Type {
		case "message":
			for _, part := range item.Content {
				if strings.TrimSpace(part.Text) != "" {
					return true
				}
			}
		case "function_call":
			if strings.TrimSpace(item.Name) != "" || strings.TrimSpace(item.Arguments) != "" || strings.TrimSpace(item.CallID) != "" {
				return true
			}
		case "reasoning":
			for _, summary := range item.Summary {
				if strings.TrimSpace(summary.Text) != "" {
					return true
				}
			}
		case "web_search_call":
			return true
		}
	}
	return false
}

func (a *bufferedResponsesAccumulator) responseSnapshot() *apicompat.ResponsesResponse {
	if a == nil || a.response == nil {
		return nil
	}
	if a.response.Status == "" {
		a.response.Status = "incomplete"
	}
	return a.response
}

func (a *bufferedResponsesAccumulator) ensureOutput(index int, outputType string) *apicompat.ResponsesOutput {
	if a.response == nil {
		a.response = &apicompat.ResponsesResponse{Object: "response", Status: "incomplete"}
	}
	if index < 0 {
		index = 0
	}
	for len(a.response.Output) <= index {
		a.response.Output = append(a.response.Output, apicompat.ResponsesOutput{})
	}
	item := &a.response.Output[index]
	if item.Type == "" {
		item.Type = outputType
	}
	return item
}

func ensureContentPart(item *apicompat.ResponsesOutput, index int) *apicompat.ResponsesContentPart {
	if item == nil {
		return nil
	}
	if index < 0 {
		index = 0
	}
	for len(item.Content) <= index {
		item.Content = append(item.Content, apicompat.ResponsesContentPart{})
	}
	return &item.Content[index]
}

func ensureSummaryPart(item *apicompat.ResponsesOutput, index int) *apicompat.ResponsesSummary {
	if item == nil {
		return nil
	}
	if index < 0 {
		index = 0
	}
	for len(item.Summary) <= index {
		item.Summary = append(item.Summary, apicompat.ResponsesSummary{})
	}
	return &item.Summary[index]
}
