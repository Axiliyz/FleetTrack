package validator

import (
	"strings"
	"testing"
)

// TestIsNameValid проверяет, что длина имени считается в символах, а не в байтах
func TestIsNameValid(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"латиница", "Ivan Petrov", true},
		{"длинное русское имя (20 букв, 40 байт)", "Диспетчер симулятора", true},
		{"ровно 35 кириллических символов", strings.Repeat("ж", 35), true},
		{"36 кириллических символов", strings.Repeat("ж", 36), false},
		{"36 латинских символов", strings.Repeat("a", 36), false},
		{"пустое", "", false},
		{"только пробелы", "   ", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNameValid(tt.in); got != tt.want {
				t.Errorf("IsNameValid(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestIsNumberPlateValid проверяет русские номера в обоих алфавитах
func TestIsNumberPlateValid(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"А123ВС77", true},
		{"А123ВС177", true},
		{"A123BC77", true},
		{"А123ВС", false},
		{"А123ВС1777", false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := IsNumberPlateValid(tt.in); got != tt.want {
				t.Errorf("IsNumberPlateValid(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// TestIsVINValid проверяет, что VIN — 17 заглавных латинских букв и цифр
func TestIsVINValid(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"обычный", "1HGCM82633A123456", true},
		{"17 байт кириллицы и цифры", "ВВВВВВВВ1", false},
		{"17 кириллических символов", strings.Repeat("В", 17), false},
		{"строчные", "1hgcm82633a123456", false},
		{"с дефисом", "1HGCM-2633A123456", false},
		{"короткий", "1HGCM82633A12345", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsVINValid(tt.in); got != tt.want {
				t.Errorf("IsVINValid(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
