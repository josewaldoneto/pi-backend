package firebase

import (
	"context"
	"fmt"
	"strconv" // Para converter workspaceID int64 para string

	"cloud.google.com/go/firestore"
	"google.golang.org/api/iterator" // Para iterator.Done
	// "projeto-integrador/utilities" // Se for usar seu logger aqui
)

// Consistente com o usado nos handlers de tarefas

// DeleteWorkspaceAndSubcollectionsFromFirestore deleta um documento de workspace e todas as suas subcoleções (como tarefas).
// workspaceIDPg é o ID numérico do workspace no PostgreSQL, que é usado como ID do documento no Firestore.
func DeleteWorkspaceAndSubcollectionsFromFirestore(tasksSubCollectionName string, ctx context.Context, client *firestore.Client, workspaceIDPg int64) error {
	workspaceDocIDStr := strconv.FormatInt(workspaceIDPg, 10)
	workspaceRef := client.Collection("workspaces").Doc(workspaceDocIDStr)

	// 1. Deletar a subcoleção de tarefas
	// Para deletar uma subcoleção, você precisa deletar todos os seus documentos.
	// O Firestore não deleta subcoleções automaticamente ao deletar o documento pai.
	tasksRef := workspaceRef.Collection(tasksSubCollectionName)
	batchSize := 500 // O Firestore recomenda batches de até 500 operações

	for {
		// Obter um batch de documentos da subcoleção de tarefas
		iter := tasksRef.Limit(batchSize).Documents(ctx)
		numDeleted := 0

		// Criar um novo batch para as operações de deleção
		batch := client.Batch()
		for {
			doc, err := iter.Next()
			if err == iterator.Done {
				break // Fim dos documentos neste batch de leitura
			}
			if err != nil {
				// utilities.LogError(err, fmt.Sprintf("Erro ao iterar documentos da subcoleção de tarefas para workspace %s", workspaceDocIDStr))
				return fmt.Errorf("erro ao iterar tarefas para deleção no workspace %s: %w", workspaceDocIDStr, err)
			}
			batch.Delete(doc.Ref) // Adiciona a operação de deleção ao batch
			numDeleted++
		}

		// Se não houver mais documentos para deletar, sair do loop principal
		if numDeleted == 0 {
			break
		}

		// Comitar o batch de deleções
		_, err := batch.Commit(ctx)
		if err != nil {
			// utilities.LogError(err, fmt.Sprintf("Erro ao comitar batch de deleção de tarefas para workspace %s", workspaceDocIDStr))
			return fmt.Errorf("erro ao deletar batch de tarefas no workspace %s: %w", workspaceDocIDStr, err)
		}
		// utilities.LogInfo(fmt.Sprintf("Deletadas %d tarefas do workspace %s no Firestore.", numDeleted, workspaceDocIDStr))
	}
	// utilities.LogInfo(fmt.Sprintf("Todas as tarefas da subcoleção do workspace %s foram deletadas do Firestore.", workspaceDocIDStr))

	// 2. Deletar o documento principal do workspace
	_, err := workspaceRef.Delete(ctx)
	if err != nil {
		// utilities.LogError(err, fmt.Sprintf("Erro ao deletar documento do workspace %s do Firestore", workspaceDocIDStr))
		// Mesmo que as tarefas tenham sido deletadas, o documento do workspace pode não ter sido.
		// Pode ser um erro de "não encontrado" se já foi deletado, o que pode ser OK.
		// Para simplificar, retornamos o erro. Em produção, pode querer checar se é "not found".
		return fmt.Errorf("erro ao deletar documento do workspace %s do Firestore: %w", workspaceDocIDStr, err)
	}
	// utilities.LogInfo(fmt.Sprintf("Documento do workspace %s deletado com sucesso do Firestore.", workspaceDocIDStr))

	return nil
}
