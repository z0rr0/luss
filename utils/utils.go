// package utils contains LUSS aditional methods.
package utils

import (
    "github.com/z0rr0/luss/conf"
    "github.com/z0rr0/luss/db"
)

var (
    // Mode is a current debug/release mode.
    Mode = ReleaseMode
    // LoggerError is a logger for error messages
    LoggerError = log.New(os.Stderr, "ERROR [LUSS]: ", log.Ldate|log.Ltime|log.Lshortfile)
    // LoggerInfo is a logger for info messages
    LoggerInfo = log.New(os.Stdout, "INFO [LUSS]: ", log.Ldate|log.Ltime|log.Lshortfile)
    // LoggerDebug is a logger for debug messages
    LoggerDebug = log.New(ioutil.Discard, "DEBUG [LUSS]: ", log.Ldate|log.Lmicroseconds|log.Llongfile)
    // Configuration is a main configuration object. It is used as a singleton.
    Configuration *conf.Config
    // Conns is a channel to get database connection.
    Conns chan db.Conn
)

// Debug activates debug mode.
func Debug(debug bool) {
    debugHandler := ioutil.Discard
    if debug {
        debugHandler = os.Stdout
        Mode = DebugMode
    } else {
        Mode = ReleaseMode
    }
    LoggerDebug = log.New(debugHandler, "DEBUG [LUSS]: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func checkDb() {

}
