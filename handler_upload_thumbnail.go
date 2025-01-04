package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/database"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = (10 << 20)

	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't Parse multipart form", err)
		return
	}

	file, header, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't find the thumbnail", err)
		return
	}
	defer file.Close()

	ct := header.Header.Get("Content-Type")
	fileContents, err := io.ReadAll(file)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to read in the file", err)
		return
	}
	splitContentType := strings.Split(ct, "/")
	if len(splitContentType) < 2 {
		respondWithError(w, http.StatusInternalServerError, "Invalid content type", nil)
		return
	}
	fileExt := splitContentType[1]

	meta, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "User does not own the video", err)
		return
	}
	/*tn := thumbnail{
		data:      filecontents,
		mediaType: ct,
	}
	videoThumbnails[meta.ID] = tn*/

	baseUrl := "http://localhost:" + cfg.port
	vUrl := baseUrl + "video/" + videoID.String()

	//thUrl := baseUrl + "thumbnails/" + videoID.String()
	fileLocation := cfg.assetsRoot + "/" + videoID.String() + "." + fileExt
	imgFile, err := os.Create(fileLocation)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create the image file", err)
		return
	}
	defer imgFile.Close()
	reader := bytes.NewReader(fileContents)
	_, err = io.Copy(imgFile, reader)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Unable to process the byte reader", err)
		return
	}

	thUrl := baseUrl + "/" + cfg.assetsRoot + "/" + videoID.String() + "." + fileExt
	videoItem := database.Video{
		ID:           meta.ID,
		CreatedAt:    meta.CreatedAt,
		UpdatedAt:    time.Now(),
		VideoURL:     &vUrl,
		ThumbnailURL: &thUrl,
		CreateVideoParams: database.CreateVideoParams{
			Title:       meta.Title,
			Description: meta.Description,
			UserID:      userID,
		},
	}
	err = cfg.db.UpdateVideo(videoItem)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Update call failed", err)
		return
	}
	respondWithJSON(w, http.StatusOK, videoItem)
}
