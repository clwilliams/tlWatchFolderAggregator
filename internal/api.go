package internal

import (
	"encoding/json"
	"net/http"
	"strconv"

	log "github.com/rs/zerolog/log"

	"github.com/tlWatchFolderAggregator/elasticSearch"
)

// GetAll returns a list of articles
func GetAll(config *elasticSearch.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corsResponseHeader(w, false)

		fsNodes, totalHits, err := config.GetAllFsNodes()
		if err != nil {
			log.Fatal().Err(err).Msg("Failed to retrieve all the documents from elastic search")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		js, err := json.Marshal(fsNodes)
		if err != nil {
			log.Fatal().Err(err).Msg("Error marshalling elastic search documents")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		corsResponseHeaderTotalCount(w, totalHits)
		w.Write(js)
	})
}

// GetFsNodesForWatchFolder returns a list of articles
func GetFsNodesForWatchFolder(config *elasticSearch.App) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		corsResponseHeader(w, false)

		folder, ok := r.URL.Query()["folder"]
		if !ok {
			http.Error(w, "folder argument must be set", http.StatusInternalServerError)
		}
		fsNodes, totalHits, err := config.GetFsNodesForWatchFolder(folder[0])
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		js, err := json.Marshal(fsNodes)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		corsResponseHeaderTotalCount(w, totalHits)
		w.Write(js)
	})
}

// all responses need this set when fulfilling the request
func corsResponseHeader(w http.ResponseWriter, includeTimeout bool) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type")
	if includeTimeout {
		w.Header().Set("Access-Control-Max-Age", "1728000")
	}
}

// responses returning multiple items need this when fulfilling the response
func corsResponseHeaderTotalCount(w http.ResponseWriter, count int64) {
	w.Header().Set("Access-Control-Expose-Headers", "X-Total-Count")
	w.Header().Set("X-Total-Count", strconv.FormatInt(count, 10))
}
