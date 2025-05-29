package ai_service // ou handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil" // Para Go < 1.16, ou io.ReadAll para Go >= 1.16
	"net/http"
	"os" // Para ler a API Key da IA de variáveis de ambiente
	"time"
	// "projeto-integrador/utilities" // Se for usar seu logger
)

const aiApiBaseURL = "URL_BASE_DA_SUA_API_DE_IA" // Ex: "https://api.provedor-ia.com/v1"
const aiApiTimeout = 30 * time.Second

// CallAIAPI faz uma requisição POST para a API de IA.
// aiEndpointPath: ex: "/code-review", "/summarize"
// requestPayload: o corpo da requisição a ser enviado como JSON.
// targetResponse: um ponteiro para a struct onde a resposta JSON bem-sucedida será decodificada.
// targetErrorResponse: um ponteiro para a struct onde uma resposta de erro JSON da API de IA será decodificada.
func CallAIAPI(ctx context.Context, aiEndpointPath string, requestPayload interface{}, targetResponse interface{}, targetErrorResponse interface{}) (int, error) {
	aiAPIKey := os.Getenv("AI_API_KEY") // Obtenha sua API Key da IA de uma variável de ambiente

	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		// utilities.LogError(err, "Erro ao fazer marshal do payload para API de IA")
		return 0, fmt.Errorf("erro ao preparar dados para API de IA: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", aiApiBaseURL+aiEndpointPath, bytes.NewBuffer(jsonData))
	if err != nil {
		// utilities.LogError(err, "Erro ao criar requisição para API de IA")
		return 0, fmt.Errorf("erro ao criar requisição para API de IA: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if aiAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+aiAPIKey) // Ou outro esquema de Auth que sua API de IA use
	}

	client := &http.Client{Timeout: aiApiTimeout}
	resp, err := client.Do(req)
	if err != nil {
		// utilities.LogError(err, "Erro ao enviar requisição para API de IA")
		return 0, fmt.Errorf("erro ao comunicar com API de IA: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := ioutil.ReadAll(resp.Body) // ou io.ReadAll para Go 1.16+
	if err != nil {
		// utilities.LogError(err, "Erro ao ler corpo da resposta da API de IA")
		return resp.StatusCode, fmt.Errorf("erro ao ler resposta da API de IA: %w", err)
	}

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		if targetResponse != nil {
			if err := json.Unmarshal(bodyBytes, targetResponse); err != nil {
				// utilities.LogError(err, "Erro ao fazer unmarshal da resposta de sucesso da API de IA")
				return resp.StatusCode, fmt.Errorf("erro ao processar resposta de sucesso da API de IA: %w", err)
			}
		}
	} else {
		// Tenta decodificar para a struct de erro esperada da API de IA
		if targetErrorResponse != nil {
			// utilities.LogWarn(fmt.Sprintf("API de IA retornou status %d. Corpo: %s", resp.StatusCode, string(bodyBytes)))
			if err := json.Unmarshal(bodyBytes, targetErrorResponse); err != nil {
				// Se não conseguir decodificar para a struct de erro, pode ser um erro diferente
				// utilities.LogWarn(fmt.Sprintf("Não foi possível fazer unmarshal da resposta de erro da API de IA no formato esperado: %v", err))
			}
		}
		// Retorna um erro genérico com o status code e o corpo para debug
		return resp.StatusCode, fmt.Errorf("API de IA retornou status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return resp.StatusCode, nil
}
