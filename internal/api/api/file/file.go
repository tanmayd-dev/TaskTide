package file

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/consts"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
	"github.com/tanmayd-dev/TaskTide/internal/repositories"
	"github.com/tanmayd-dev/TaskTide/internal/utils"
)

func addFile(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseMultipartForm(32 * int64(consts.MB))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse form", http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not read 'file'", http.StatusBadRequest)
			return
		}
		defer file.Close()

		minioFile, err := repositories.UploadFile(queries, ctx, minioClient, "files", header.Header.Get("ContentType"), file, header.Filename, header.Size)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not create file", http.StatusBadRequest)
			return
		}

		utils.WriteJSON(w, http.StatusOK, minioFile)
	}
}

func getFile(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var fileModel models.InputFile
		err := json.NewDecoder(r.Body).Decode(&fileModel)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "could not parse input", http.StatusBadRequest)
			return
		}
		minioFile, info, err := repositories.GetFile(ctx, queries, fileModel.ID, minioClient, os.Getenv("FILES_BUCKET_NAME"))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "could not get file", http.StatusBadRequest)
			return
		}
		defer minioFile.Close()

		w.Header().Set(
			"Content-Type",
			info.ContentType,
		)

		w.Header().Set(
			"Content-Length",
			strconv.FormatInt(info.Size, 10),
		)

		_, err = io.Copy(w, minioFile)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}
}

func addFileToWorkflow(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var fileModel models.InputFile
		err := json.NewDecoder(r.Body).Decode(&fileModel)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "could not parse input", http.StatusBadRequest)
			return
		}
		err = repositories.AttachFile(queries, ctx, minioClient, fileModel.ID, fileModel.WorkflowID)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not attach file", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", addFile(queries, ctx, minioClient))
		r.Get("/", getFile(queries, ctx, minioClient))
		r.Post("/workflow", addFileToWorkflow(queries, ctx, minioClient))
	}
}
