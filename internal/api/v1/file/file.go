package file

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/minio/minio-go/v7"
	"github.com/tanmayd-dev/TaskTide/internal/consts"
	"github.com/tanmayd-dev/TaskTide/internal/db"
	"github.com/tanmayd-dev/TaskTide/internal/models"
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

		exists, err := minioClient.BucketExists(ctx, "files")
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not verify bucket existance", http.StatusBadRequest)
			return
		}

		if !exists {
			err = minioClient.MakeBucket(ctx, "files", minio.MakeBucketOptions{})
			if err != nil {
				log.Printf("%v\n", err.Error())
				http.Error(w, "Could not create bucket", http.StatusBadRequest)
				return
			}
		}

		minioFile, err := queries.CreateFile(
			ctx,
			pgtype.Text{
				String: header.Filename,
				Valid:  true,
			},
		)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not write to database", http.StatusBadRequest)
			return
		}

		_, err = minioClient.PutObject(
			ctx,
			"files",
			fmt.Sprintf("%v/%v", minioFile.ID, header.Filename),
			file,
			header.Size,
			minio.PutObjectOptions{
				ContentType: header.Header.Get("ContentType"),
			},
		)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not write to object store", http.StatusBadRequest)

			err := queries.DeleteFile(ctx, minioFile.ID)
			if err != nil {
				log.Printf("%v\n", err.Error())
				http.Error(w, "We wrote you object but lost track of it. It is so over", http.StatusInternalServerError)
			}

			return
		}
		json.NewEncoder(w).Encode(minioFile)
	}
}

func getFile(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var file models.File
		err := json.NewDecoder(r.Body).Decode(&file)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse input", http.StatusBadRequest)
			return
		}
		UUID, err := uuid.Parse(string(file.ID))
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not parse UUID", http.StatusBadRequest)
			return
		}
		filesData, err := queries.GetFile(ctx, pgtype.UUID{
			Bytes: [16]byte(UUID),
			Valid: true,
		})
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, "Could not contact Database", http.StatusBadRequest)
			return
		}
		if len(filesData) == 0 {
			http.Error(w, "Could not get file from DB", http.StatusBadRequest)
			return
		}
		fileData := filesData[0]
		MinioID := fmt.Sprintf("%v/%v", file.ID, fileData.FileName.String)
		minioFile, err := minioClient.GetObject(
			ctx,
			"files",
			MinioID,
			minio.GetObjectOptions{},
		)
		if err != nil {
			log.Printf("%v\n", err.Error())
			http.Error(w, fmt.Sprintf("Could not get file %v", MinioID), http.StatusBadRequest)
			return
		}
		defer minioFile.Close()

		info, err := minioFile.Stat()
		if err != nil {
			http.Error(w, err.Error(), 404)
			return
		}

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

func Router(queries *db.Queries, ctx context.Context, minioClient *minio.Client) func(chi.Router) {
	return func(r chi.Router) {
		r.Post("/", addFile(queries, ctx, minioClient))
		r.Get("/", getFile(queries, ctx, minioClient))
	}
}
