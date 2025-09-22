package main

import (
	"flag"
	"godab/api"
	"godab/config"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	if !api.DirExists(config.GetDownloadLocation()) {
		log.Fatalf("You must provide a valid DOWNLOAD_LOCATION folder")
	}

	asciiArt := `
  ____           _       _     
 / ___| ___   __| | __ _| |__  
| |  _ / _ \ / _\` + "`" + ` |/ _\` + "`" + ` | '_ \ 
| |_| | (_) | (_| | (_| | |_) |
 \____|\___/ \__,_|\__,_|_.__/ 
`

	var (
		album  string
		track  string
		artist string
		server bool
		port   int
	)

	flag.StringVar(&album, "album", "", "Album URL to download")
	flag.StringVar(&track, "track", "", "Track URL to download")
	flag.StringVar(&artist, "artist", "", "Artist URL to download")
	flag.BoolVar(&server, "server", false, "Start in web server mode")
	flag.IntVar(&port, "port", 8080, "Port for web server mode")
	flag.Parse()

	// Server mode takes precedence
	if server {
		startWebServer(port)
		return
	}

	// Existing CLI validation
	if album == "" && track == "" && artist == "" {
		flag.Usage()
		return
	}

	if (album != "" && track != "") || (artist != "" && track != "") || (album != "" && artist != "") {
		log.Fatalf("You can download only one between `album` and `track` at a time.")
		flag.Usage()
	}

	// fmt.Println(asciiArt)
	api.PrintColor(api.COLOR_BLUE, "%s", asciiArt)

	if album != "" {
		album, err := api.NewAlbum(album)

		if err != nil {
			log.Fatalf("Error: %s", err)
		}

		if err := album.Download(true); err != nil {
			log.Fatalf("Cannot download album %s: %s", album.Title, err)
		}
	} else if track != "" {
		track, err := api.NewTrack(track)

		if err != nil {
			log.Fatalf("Error: %s", err)
		}

		if err := track.Download(); err != nil {
			log.Fatalf("Cannot download track %s: %s", track.Title, err)
		}
	} else if artist != "" {
		artist, err := api.NewArtist(artist)

		if err != nil {
			log.Fatalf("Error: %s", err)
		}

		if err := artist.Download(); err != nil {
			log.Fatalf("Cannot download artist %s: %s", artist.Name, err)
		}

	}

	log.Println()
}

// startWebServer initializes and starts the HTTP server
func startWebServer(port int) {
	// Set Gin to release mode for production
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.DebugMode)
	}

	r := gin.Default()

	// CORS configuration
	corsOrigin := os.Getenv("CORS_ORIGIN")
	if corsOrigin == "" {
		corsOrigin = "http://localhost:3000" // Default for React dev
	}

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{corsOrigin}
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization"}
	r.Use(cors.New(corsConfig))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "crescendo",
			"version":   "1.0.0",
			"timestamp": time.Now().Unix(),
		})
	})

	// API routes group
	apiGroup := r.Group("/api")
	{
		apiGroup.GET("/status", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "Crescendo API is running",
				"download_location": config.GetDownloadLocation(),
			})
		})

		// Search endpoint
		apiGroup.GET("/search", func(c *gin.Context) {
			query := c.Query("q")
			searchType := c.DefaultQuery("type", "track") // Default to track search

			if query == "" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "query parameter 'q' is required",
				})
				return
			}

			// Validate search type
			if searchType != "track" && searchType != "album" {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "type parameter must be 'track' or 'album'",
				})
				return
			}

			// Perform search using existing API function
			results, err := api.Search(&query, searchType)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "search failed",
					"details": err.Error(),
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"query": query,
				"type": searchType,
				"results": results,
			})
		})
	}

	portStr := strconv.Itoa(port)
	serverPort := os.Getenv("SERVER_PORT")
	if serverPort != "" {
		portStr = serverPort
	}

	api.PrintColor(api.COLOR_BLUE, `
  ____                                    _
 / ___|_ __ ___  ___  ___ ___ _ __   __| | ___
| |   | '__/ _ \/ __|/ __/ _ \ '_ \ / _` + "`" + ` |/ _ \
| |___| | |  __/\__ \ (_|  __/ | | | (_| | (_) |
 \____|_|  \___||___/\___\___|_| |_|\__,_|\___/

`)

	log.Printf("🚀 Crescendo web server starting on port %s", portStr)
	log.Printf("📁 Download location: %s", config.GetDownloadLocation())
	log.Printf("🌐 Health check: http://localhost:%s/health", portStr)
	log.Printf("🔗 API status: http://localhost:%s/api/status", portStr)

	if err := r.Run(":" + portStr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
