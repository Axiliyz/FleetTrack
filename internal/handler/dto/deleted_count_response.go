package dto

// DeletedCountResponse - ответ массового удаления: сколько записей удалено
type DeletedCountResponse struct {
	Deleted int64 `json:"deleted"`
}
