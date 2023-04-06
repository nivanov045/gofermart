package log

import (
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

type Level zerolog.Level

const (
	ErrorLevel Level = Level(zerolog.ErrorLevel)
	WarnLevel  Level = Level(zerolog.WarnLevel)
	InfoLevel  Level = Level(zerolog.InfoLevel)
	DebugLevel Level = Level(zerolog.DebugLevel)
)

func Init(level Level) {
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.SetGlobalLevel(zerolog.Level(level))
}

func Panic(err error) {
	log.Error().Stack().Err(errors.Wrap(err, "from error")).Send()
	panic(err)
}

func Error(err error) {
	log.Error().Stack().Err(errors.Wrap(err, "from error")).Send()
}

func Warn(err error) {
	log.Warn().Stack().Err(errors.Wrap(err, "from error")).Send()
}

func Info(msg string) {
	log.Info().Stack().Msg(msg)
}

func Debug(msg string) {
	log.Debug().Stack().Msg(msg)
}
