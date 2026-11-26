module project

go 1.27.3

require github.com/gabriel-vasile/mimetype v1.4.16 // indirect

require internal/system v1.0.0

require internal/api/client v1.0.0

require internal/matrix/auth v1.0.0

replace internal/system => ./src/internal/system

replace internal/api/client => ./src/internal/api

replace internal/matrix/auth => ./src/internal/matrix/auth
