package domain

import "errors"

var (
	ErrBlankField                = errors.New("required field is blank")
	ErrInvalidCollection         = errors.New("collection contains invalid values")
	ErrInvalidCourseLanguage     = errors.New("invalid course language")
	ErrInvalidCourseStatus       = errors.New("invalid course generation status")
	ErrInvalidGenerationStatus   = errors.New("invalid generation pipeline status")
	ErrInvalidLessonType         = errors.New("invalid lesson type")
	ErrInvalidLevel              = errors.New("invalid level")
	ErrInvalidOrder              = errors.New("order must be greater than 0")
	ErrInvalidProgress           = errors.New("progress percent must be between 0 and 100")
	ErrInvalidDuration           = errors.New("duration must be greater than 0")
	ErrInvalidStatusTransition   = errors.New("invalid status transition")
	ErrDuplicateModuleOrder      = errors.New("duplicate module order")
	ErrDuplicateLessonOrder      = errors.New("duplicate lesson order")
	ErrMissingCourseContent      = errors.New("course has missing generated content")
	ErrInvalidPassword           = errors.New("le mot de passe est trop faible: au moins 8 caracteres, 1 majuscule et 1 caractere special")
	ErrInvalidPasswordHash       = errors.New("password hash invalide")
	ErrUsernameAlreadyExists     = errors.New("le nom d'utilisateur est deja utilise")
	ErrUserNotFound              = errors.New("utilisateur introuvable")
	ErrInvalidUsername           = errors.New("le nom doit faire entre 3 et 50 caracteres")
	ErrNewUUIDCreation           = errors.New("une erreur est survenue dans la creation de l'UUID")
	ErrInvalidClarification      = errors.New("invalid clarification question")
	ErrGenerationRequestNotReady = errors.New("generation request is not ready for this transition")
)
