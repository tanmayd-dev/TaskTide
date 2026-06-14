package repositories

import (
	"context"

	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func AddWorker(ctx context.Context, queries *db.Queries, worker models.Worker) (models.Worker, error) {
	addedWorker, err := queries.AddWorker(ctx, db.AddWorkerParams{
		WorkerHost: worker.WorkerHost,
		WorkerPort: worker.WorkerPort,
	})
	if err != nil {
		return models.Worker{}, err
	}
	return utils.WorkerModel(addedWorker), nil
}

func GetAvailableWorker(ctx context.Context, queries *db.Queries) (models.Worker, error) {
	wDB, err := queries.GetAvailableWorker(ctx)
	if err != nil {
		return models.Worker{}, err
	}
	return utils.WorkerModel(wDB), nil
}

func AssignWorkerToTask(ctx context.Context, queries *db.Queries, workerID models.WorkerID, taskExecutionID models.TaskExecutionID) (models.TaskExecution, error) {
	workerUUID, err := utils.StringToUUID(string(workerID))
	if err != nil {
		return models.TaskExecution{}, err
	}
	taskExecutionUUID, err := utils.StringToUUID(string(taskExecutionID))
	if err != nil {
		return models.TaskExecution{}, err
	}
	te, err := queries.AssignWorkerToTask(ctx, db.AssignWorkerToTaskParams{
		TaskWorker: workerUUID,
		ID:         taskExecutionUUID,
	})
	if err != nil {
		return models.TaskExecution{}, err
	}
	return utils.TaskExecutionModel(te), nil
}

func MarkWorkerUnavailable(ctx context.Context, queries *db.Queries, workerID models.WorkerID) (models.Worker, error) {
	workerUUID, err := utils.StringToUUID(string(workerID))
	if err != nil {
		return models.Worker{}, err
	}
	wDB, err := queries.MarkWorkerUnavailable(ctx, workerUUID)
	if err != nil {
		return models.Worker{}, err
	}
	return utils.WorkerModel(wDB), nil
}

func MarkWorkerAvailable(ctx context.Context, queries *db.Queries, workerID models.WorkerID) error {
	workerUUID, err := utils.StringToUUID(string(workerID))
	if err != nil {
		return err
	}
	return queries.MarkWorkerAvailable(ctx, workerUUID)
}

func DeleteWorker(ctx context.Context, queries *db.Queries, workerID models.WorkerID) error {
	workerUUID, err := utils.StringToUUID(string(workerID))
	if err != nil {
		return err
	}
	return queries.DeleteWorker(ctx, workerUUID)
}

