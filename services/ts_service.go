// services/ts_service.go
package services

import (
	config "SimpleGit/config"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

type TSService struct {
	BaseURL string
}

type HighlightRequest struct {
	Code     string `json:"code"`
	Language string `json:"language,omitempty"`
	Filename string `json:"filename,omitempty"`
}

type HighlightResponse struct {
	Highlighted      string `json:"highlighted"`
	DetectedLanguage string `json:"detectedLanguage"`
	//Symbols          []util.Symbol `json:"symbols"`
}

func NewTSService() *TSService {
	return &TSService{
		BaseURL: config.GlobalConfig.TSServiceURL,
	}
}

func (s *TSService) Highlight(code, language, filename string) (*HighlightResponse, error) {
	req := HighlightRequest{
		Code:     code,
		Language: language,
		Filename: filename,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := http.Post(s.BaseURL+"/highlight", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service returned status %d", resp.StatusCode)
	}

	var result HighlightResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
