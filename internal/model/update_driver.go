package model

// UpdateDriver указывает изменяемые поля водителя.
// Организацию водителя поменять нельзя: такого поля здесь нет намеренно
type UpdateDriver struct {
	Name *string
}
