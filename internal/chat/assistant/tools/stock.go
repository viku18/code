package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"errors"
	"net/http"
	"net/url"
	//"os"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
)

type StockTool struct {
	apiKey string
}

func NewStockTool(apiKey string) *StockTool {
	return &StockTool{
		apiKey: apiKey,
	}
}

func (s *StockTool) Name() string {
	return "get_stock_price"
}

func (s *StockTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        s.Name(),
		Description: openai.String("Get current stock price and information for a given stock symbol or company name"),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"symbol": map[string]string{
					"type":        "string",
					"description": "Stock symbol (e.g., AAPL, GOOGL, MSFT) or company name",
				},
				"detailed": map[string]string{
					"type":        "boolean",
					"description": "Whether to return detailed information including daily change, volume, etc. Defaults to false.",
				},
			},
			"required": []string{"symbol"},
		},
	})
}

func (s *StockTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	if s.apiKey == "" {
		return "", errors.New("stock API key not configured. Please set STOCK_API_KEY environment variable.")
	}

	var payload struct {
		Symbol   string `json:"symbol"`
		Detailed bool   `json:"detailed,omitempty"`
	}

	if err := json.Unmarshal(args, &payload); err != nil {
		return "", fmt.Errorf("failed to parse arguments: %w", err)
	}

	if payload.Symbol == "" {
		return "", errors.New("stock symbol is required")
	}

	// Try to resolve company name to symbol if needed
	symbol := strings.ToUpper(payload.Symbol)
	if len(symbol) > 5 { // Likely a company name rather than symbol
		resolvedSymbol, err := s.resolveSymbol(ctx, symbol)
		if err != nil {
			return "", fmt.Errorf("could not find stock symbol for '%s': %w", symbol, err)
		}
		symbol = resolvedSymbol
	}

	stockData, err := s.fetchStockData(ctx, symbol)
	if err != nil {
		return "", err
	}

	if payload.Detailed {
		return s.formatDetailedResponse(stockData), nil
	}

	return s.formatSimpleResponse(stockData), nil
}

// StockData represents the structure of the stock API response
type StockData struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Change    float64 `json:"change"`
	ChangePercent float64 `json:"changePercent"`
	Volume    int64   `json:"volume"`
	High      float64 `json:"high"`
	Low       float64 `json:"low"`
	Open      float64 `json:"open"`
	PreviousClose float64 `json:"previousClose"`
	Name      string  `json:"name"`
}

func (s *StockTool) fetchStockData(ctx context.Context, symbol string) (*StockData, error) {
	// Using Alpha Vantage API as an example - you can replace with any stock API
	baseURL := "https://www.alphavantage.co/query"
	
	queryParams := url.Values{}
	queryParams.Add("function", "GLOBAL_QUOTE")
	queryParams.Add("symbol", symbol)
	queryParams.Add("apikey", s.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+queryParams.Encode(), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse Alpha Vantage response
	var apiResponse map[string]map[string]string
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, err
	}

	quote, exists := apiResponse["Global Quote"]
	if !exists {
		return nil, fmt.Errorf("no quote data found for symbol %s", symbol)
	}

	// Parse the response data
	stockData := &StockData{
		Symbol: symbol,
	}

	// Helper function to parse float from string
	parseFloat := func(s string) float64 {
		var f float64
		fmt.Sscanf(s, "%f", &f)
		return f
	}

	// Helper function to parse int from string
	parseInt := func(s string) int64 {
		var i int64
		fmt.Sscanf(s, "%d", &i)
		return i
	}

	stockData.Price = parseFloat(quote["05. price"])
	stockData.Change = parseFloat(quote["09. change"])
	stockData.ChangePercent = parseFloat(quote["10. change percent"])
	stockData.Volume = parseInt(quote["06. volume"])
	stockData.High = parseFloat(quote["03. high"])
	stockData.Low = parseFloat(quote["04. low"])
	stockData.Open = parseFloat(quote["02. open"])
	stockData.PreviousClose = parseFloat(quote["08. previous close"])

	// Get company name from another API call
	if name, err := s.getCompanyName(ctx, symbol); err == nil {
		stockData.Name = name
	}

	return stockData, nil
}

func (s *StockTool) getCompanyName(ctx context.Context, symbol string) (string, error) {
	baseURL := "https://www.alphavantage.co/query"
	
	queryParams := url.Values{}
	queryParams.Add("function", "OVERVIEW")
	queryParams.Add("symbol", symbol)
	queryParams.Add("apikey", s.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+queryParams.Encode(), nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var overview map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&overview); err != nil {
		return "", err
	}

	if name, exists := overview["Name"].(string); exists {
		return name, nil
	}

	return symbol, nil
}

func (s *StockTool) resolveSymbol(ctx context.Context, query string) (string, error) {
	baseURL := "https://www.alphavantage.co/query"
	
	queryParams := url.Values{}
	queryParams.Add("function", "SYMBOL_SEARCH")
	queryParams.Add("keywords", query)
	queryParams.Add("apikey", s.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", baseURL+"?"+queryParams.Encode(), nil)
	if err != nil {
		return "", err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var searchResponse struct {
		BestMatches []struct {
			Symbol      string `json:"1. symbol"`
			Name        string `json:"2. name"`
			MatchScore  string `json:"9. matchScore"`
		} `json:"bestMatches"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return "", err
	}

	if len(searchResponse.BestMatches) == 0 {
		return "", fmt.Errorf("no matching symbols found for '%s'", query)
	}

	// Return the best match
	return searchResponse.BestMatches[0].Symbol, nil
}

func (s *StockTool) formatSimpleResponse(data *StockData) string {
	changeIndicator := "▲"
	if data.Change < 0 {
		changeIndicator = "▼"
	}

	if data.Name != "" {
		return fmt.Sprintf("%s (%s): $%.2f %s%.2f (%.2f%%)", 
			data.Name, data.Symbol, data.Price, changeIndicator, 
			abs(data.Change), abs(data.ChangePercent))
	}

	return fmt.Sprintf("%s: $%.2f %s%.2f (%.2f%%)", 
		data.Symbol, data.Price, changeIndicator, 
		abs(data.Change), abs(data.ChangePercent))
}

func (s *StockTool) formatDetailedResponse(data *StockData) string {
	changeIndicator := "▲"
	if data.Change < 0 {
		changeIndicator = "▼"
	}

	response := fmt.Sprintf("Stock: %s (%s)\n", data.Name, data.Symbol)
	response += fmt.Sprintf("Price: $%.2f\n", data.Price)
	response += fmt.Sprintf("Change: %s$%.2f (%.2f%%)\n", changeIndicator, abs(data.Change), abs(data.ChangePercent))
	response += fmt.Sprintf("Open: $%.2f\n", data.Open)
	response += fmt.Sprintf("High: $%.2f\n", data.High)
	response += fmt.Sprintf("Low: $%.2f\n", data.Low)
	response += fmt.Sprintf("Previous Close: $%.2f\n", data.PreviousClose)
	response += fmt.Sprintf("Volume: %s\n", formatVolume(data.Volume))

	return response
}

// Helper function to get absolute value of float64
func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// Helper function to format volume with commas
func formatVolume(volume int64) string {
	if volume == 0 {
		return "0"
	}
	
	str := fmt.Sprintf("%d", volume)
	n := len(str)
	if n <= 3 {
		return str
	}
	
	var result strings.Builder
	for i, r := range str {
		if i > 0 && (n-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(r)
	}
	return result.String()
}