package ai_services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil" // ou "io" se estiver usando Go 1.16+
	"net/http"
	"projeto-integrador/utilities" // Onde você definiu LogError, LogWarn, etc.
	"time"
)

// No seu pacote ai_service ou handlers (onde CallAIAPI está)
// ... (imports) ...
const aiApiBaseURL = "https://servico-ia.onrender.com"
const aiApiTimeout = 45 * time.Second

func CallAIAPI(ctx context.Context, aiEndpointPath string, requestPayload interface{}, targetSuccessResponse interface{}) (statusCode int, rawResponseBody []byte, err error) {
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return 0, nil, fmt.Errorf("erro ao preparar dados para API de IA: %w", err)
	}

	fullURL := aiApiBaseURL + aiEndpointPath
	req, err := http.NewRequestWithContext(ctx, "POST", fullURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return 0, nil, fmt.Errorf("erro ao criar requisição para API de IA: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: aiApiTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, fmt.Errorf("erro ao comunicar com API de IA: %w", err)
	}
	defer resp.Body.Close()

	rawResponseBody, err = ioutil.ReadAll(resp.Body) // ou io.ReadAll para Go 1.16+
	if err != nil {
		return resp.StatusCode, nil, fmt.Errorf("erro ao ler resposta da API de IA: %w", err)
	}

	statusCode = resp.StatusCode

	if statusCode >= 200 && statusCode < 300 {
		if targetSuccessResponse != nil && rawResponseBody != nil {
			if umErr := json.Unmarshal(rawResponseBody, targetSuccessResponse); umErr != nil {
				// Logar o erro de unmarshal, mas não necessariamente tratar como falha da chamada à IA
				utilities.LogInfo(fmt.Sprintf("CallAIAPI: Erro ao fazer unmarshal da resposta de sucesso da API de IA em targetSuccessResponse: %v. Corpo: %s", umErr, string(rawResponseBody)))
				// Opcional: retornar um erro específico aqui se o unmarshal for crítico
			}
		}
		return statusCode, rawResponseBody, nil // Sucesso
	}

	// Erro da API de IA (status code não é 2xx)
	return statusCode, rawResponseBody, fmt.Errorf("API de IA (%s) retornou status %d", fullURL, statusCode)
}
