package validator

import (
	"strings"
	"unicode/utf8"
)

// Длины считаются в символах, а не в байтах: кириллическая буква в UTF-8 занимает два байта,
// и len() отвергал бы русское имя из 18 букв или номер «А123ВС77». Колонки VARCHAR(n)
// в Postgres тоже считают символы, так что проверка совпадает с тем, что примет база
const (
	maxNameLen        = 35 // users.name, drivers.name — VARCHAR(35)
	minNumberPlateLen = 8  // А123ВС77
	maxNumberPlateLen = 9  // А123ВС177
	vinLen            = 17
)

// IsNameValid проверяет, что имя не пустое и не длиннее maxNameLen символов
func IsNameValid(name string) bool {
	return strings.TrimSpace(name) != "" && utf8.RuneCountInString(name) <= maxNameLen
}

// IsNumberPlateValid проверяет длину госномера в символах: латиницей или кириллицей
func IsNumberPlateValid(plate string) bool {
	n := utf8.RuneCountInString(plate)
	return n >= minNumberPlateLen && n <= maxNumberPlateLen
}

// IsVINValid проверяет, что VIN — ровно 17 заглавных латинских букв и цифр.
// Раньше проверялась длина в байтах, и «ВВВВВВВВ1» (17 байт кириллицы) проходил как VIN
func IsVINValid(vin string) bool {
	if len(vin) != vinLen {
		return false
	}
	for _, r := range vin {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}
