package flows

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/firebase/genkit/go/ai"
	"github.com/firebase/genkit/go/genkit"
)

// EmailRequestDetails define a estrutura de entrada para o flow de geração de e-mail.
type EmailRequestDetails struct {
	Destinatario       string   `json:"destinatario"`
	Remetente          string   `json:"remetente"`
	AssuntoProposto    string   `json:"assuntoProposto"`
	PontosChave        []string `json:"pontosChave"`
	Tom                string   `json:"tom"`
	Idioma             string   `json:"idioma"`
	ModeloPreferencial string   `json:"modeloPreferencial,omitempty"` // Opcional, para usar um modelo específico
}

// EmailResponse define a estrutura de saída.
type EmailResponse struct {
	CorpoEmail string `json:"corpoEmail"`
}

// Definindo o Flow para Gerar E-mails
// Usaremos "emailGenerator" como o nome do flow.
var EmailGeneratorFlow = genkit.DefineFlow(
	"emailGenerator", // Nome do Flow
	func(ctx context.Context, req EmailRequestDetails) (*EmailResponse, error) {
		if req.Destinatario == "" || req.Remetente == "" || req.AssuntoProposto == "" || len(req.PontosChave) == 0 {
			return nil, errors.New("destinatario, remetente, assuntoProposto e pontosChave são obrigatórios")
		}

		if req.Tom == "" {
			req.Tom = "profissional" // Valor padrão
		}
		if req.Idioma == "" {
			req.Idioma = "português" // Valor padrão
		}

		// Constrói a lista de pontos chave para o prompt
		var pontosChaveStr strings.Builder
		for _, ponto := range req.PontosChave {
			pontosChaveStr.WriteString(fmt.Sprintf("- %s\n", ponto))
		}

		// Monta o prompt para o modelo Gemini
		prompt := fmt.Sprintf(`
Por favor, escreva um e-mail em %s.

**De:** %s
**Para:** %s
**Assunto Sugerido (você pode refinar se necessário):** %s
**Tom do e-mail:** %s

**Pontos principais a serem incluídos no corpo do e-mail:**
%s
**Instruções Adicionais:**
- Certifique-se de que o e-mail flua bem e seja coeso.
- Comece com uma saudação apropriada e termine com uma despedida cordial.
- Não inclua o "De:", "Para:" ou "Assunto:" no corpo do e-mail gerado, apenas o conteúdo do e-mail em si.
- O e-mail deve estar pronto para ser enviado.

**Corpo do E-mail:**
`, req.Idioma, req.Remetente, req.Destinatario, req.AssuntoProposto, req.Tom, pontosChaveStr.String())

		// Escolhe o modelo a ser usado
		// Se um modelo preferencial foi especificado na requisição, usa ele.
		// Caso contrário, o Genkit usará o modelo padrão configurado na inicialização.
		var model *ai.Model
		if req.ModeloPreferencial != "" {
			m := ai.LookupModel(ctx, req.ModeloPreferencial)
			if m == nil {
				return nil, fmt.Errorf("modelo preferencial '%s' não encontrado", req.ModeloPreferencial)
			}
			model = m
		} else {
			// Se nenhum modelo preferencial for especificado,
			// podemos tentar obter o padrão global ou um específico do plugin.
			// Por simplicidade, vamos assumir que o modelo padrão global está configurado
			// ou podemos especificar um aqui se necessário.
			// Para usar o default global do Genkit: não precisa especificar o modelo na chamada Generate.
			// Para especificar um aqui, se não houver preferencial:
			// model = ai.LookupModel(ctx, "googleai/gemini-1.5-flash-latest") // Exemplo
		}

		fmt.Println("--- Prompt Enviado ao Gemini ---")
		fmt.Println(prompt)
		fmt.Println("-------------------------------")

		var resp *ai.GenerateResponse
		var err error

		// Prepara as opções de geração
		generateOptions := &ai.GenerateRequest{
			Prompt: ai.NewGenerateRequestSinglePart(prompt),
			Config: &ai.GenerationConfig{
				// Temperature: 0.7, // Exemplo de configuração, pode ajustar
			},
		}

		if model != nil {
			resp, err = model.Generate(ctx, generateOptions)
		} else {
			// Se 'model' for nil, o Genkit usará o modelo padrão configurado globalmente.
			resp, err = genkit.Generate(ctx, nil, generateOptions) // Passar nil para 'g' aqui para usar o Genkit global.
		}

		if err != nil {
			return nil, fmt.Errorf("falha ao gerar e-mail com o modelo: %v", err)
		}

		if len(resp.Candidates) == 0 || len(resp.Candidates[0].Message.Content) == 0 {
			// Verifica o feedback do prompt em caso de bloqueio ou outros problemas
			if resp.UsageMetadata != nil && resp.UsageMetadata.PromptFeedback != nil {
				if resp.UsageMetadata.PromptFeedback.BlockReason != "" {
					reasonMsg := resp.UsageMetadata.PromptFeedback.BlockReasonMessage
					if reasonMsg == "" {
						reasonMsg = string(resp.UsageMetadata.PromptFeedback.BlockReason)
					}
					return nil, fmt.Errorf("geração de e-mail bloqueada: %s", reasonMsg)
				}
			}
			return nil, errors.New("resposta do modelo vazia ou inválida")
		}

		corpoEmailGerado := resp.Candidates[0].Message.Content[0].Text

		fmt.Println("--- E-mail Gerado ---")
		fmt.Println(corpoEmailGerado)
		fmt.Println("--------------------")

		return &EmailResponse{CorpoEmail: corpoEmailGerado}, nil
	},
)
