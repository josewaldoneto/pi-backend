package utilities

import (
	"log"
	"os"
	"time"
)

var (
	InfoLogger  *log.Logger
	ErrorLogger *log.Logger
	DebugLogger *log.Logger
)

// InitLogger inicializa os loggers
func InitLogger() {
	// Configuração do formato de data/hora
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)

	// Criar loggers com diferentes prefixos
	InfoLogger = log.New(os.Stdout, "\033[32m[INFO]\033[0m ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	ErrorLogger = log.New(os.Stderr, "\033[31m[ERROR]\033[0m ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
	DebugLogger = log.New(os.Stdout, "\033[36m[DEBUG]\033[0m ", log.Ldate|log.Ltime|log.Lmicroseconds|log.Lshortfile)
}

// LogRequest registra informações sobre a requisição HTTP
func LogRequest(method, path, remoteAddr string, status int, duration time.Duration) {
	InfoLogger.Printf("%s %s %s %d %v", method, path, remoteAddr, status, duration)
}

// LogError registra erros com stack trace
func LogError(err error, context string) {
	ErrorLogger.Printf("%s: %v", context, err)
}

// LogDebug registra informações de debug
func LogDebug(format string, v ...interface{}) {
	DebugLogger.Printf(format, v...)
}

// LogInfo registra informações gerais
func LogInfo(format string, v ...interface{}) {
	InfoLogger.Printf(format, v...)
}
