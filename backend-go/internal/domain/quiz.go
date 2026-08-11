package domain

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type NewQuizParams struct {
	LessonID   uuid.UUID
	Type       QuizType
	Difficulty Difficulty
	Title      string
	Objective  string
	Questions  []QuizQuestion
}

type Quiz struct {
	ID          uuid.UUID
	LessonID    uuid.UUID
	Type        QuizType
	Difficulty  Difficulty
	Title       string
	Objective   string
	Questions   []QuizQuestion
	RawAIOutput json.RawMessage
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type QuizQuestion struct {
	Order      int
	Type       QuizQuestionType
	Question   string
	Options    []QuizOption
	Answer     QuizAnswer
	Correction string
}

type QuizOption struct {
	Order int
	Text  string
}

type QuizAnswer struct {
	Answer  *string
	Answers []string
}

func NewQuiz(params NewQuizParams) (Quiz, error) {
	return NewQuizAt(params, time.Now())
}

func NewQuizAt(params NewQuizParams, now time.Time) (Quiz, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return Quiz{}, fmt.Errorf("%w: %v", ErrNewUUIDCreation, err)
	}

	quiz := Quiz{
		ID:         id,
		LessonID:   params.LessonID,
		Type:       params.Type,
		Difficulty: params.Difficulty,
		Title:      normalizeText(params.Title),
		Objective:  normalizeText(params.Objective),
		Questions:  normalizeQuizQuestions(params.Questions),
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	if err := quiz.Validate(); err != nil {
		return Quiz{}, err
	}
	return quiz, nil
}

func (q Quiz) Validate() error {
	if q.ID == uuid.Nil {
		return fmt.Errorf("%w: quiz id", ErrBlankField)
	}
	if q.LessonID == uuid.Nil {
		return fmt.Errorf("%w: lesson id", ErrBlankField)
	}
	if err := q.Type.Validate(); err != nil {
		return err
	}
	if err := q.Difficulty.Validate(); err != nil {
		return err
	}
	if err := requireNotBlank("quiz title", q.Title); err != nil {
		return err
	}
	if err := requireNotBlank("quiz objective", q.Objective); err != nil {
		return err
	}
	if len(q.Questions) == 0 {
		return fmt.Errorf("%w: quiz questions", ErrInvalidCollection)
	}
	if err := validateContiguousQuizQuestionOrders(q.Questions); err != nil {
		return err
	}
	for _, question := range q.Questions {
		if err := question.Validate(); err != nil {
			return err
		}
		if err := q.validateQuestionMatchesQuizType(question); err != nil {
			return err
		}
	}
	return nil
}

func (q *Quiz) AddQuestion(question QuizQuestion) error {
	question = normalizeQuizQuestion(question)
	if err := question.Validate(); err != nil {
		return err
	}
	if err := q.validateQuestionMatchesQuizType(question); err != nil {
		return err
	}
	for _, existing := range q.Questions {
		if existing.Order == question.Order {
			return ErrDuplicateQuestionOrder
		}
	}
	nextQuestions := append(append([]QuizQuestion{}, q.Questions...), question)
	if err := validateContiguousQuizQuestionOrders(nextQuestions); err != nil {
		return err
	}
	q.Questions = append(q.Questions, question)
	q.UpdatedAt = time.Now()
	return nil
}

func (q Quiz) validateQuestionMatchesQuizType(question QuizQuestion) error {
	if q.Type == QuizTypeMixed {
		return nil
	}
	if string(q.Type) != string(question.Type) {
		return fmt.Errorf("%w: quiz type %s does not accept question type %s", ErrInvalidQuizQuestionType, q.Type, question.Type)
	}
	return nil
}

func (q QuizQuestion) Validate() error {
	if err := validatePositiveInt("quiz question order", q.Order); err != nil {
		return err
	}
	if err := q.Type.Validate(); err != nil {
		return err
	}
	if err := requireNotBlank("quiz question", q.Question); err != nil {
		return err
	}
	if err := requireNotBlank("quiz correction", q.Correction); err != nil {
		return err
	}
	if err := validateContiguousQuizOptionOrders(q.Options); err != nil {
		return err
	}
	for _, option := range q.Options {
		if err := option.Validate(); err != nil {
			return err
		}
	}
	return q.Answer.ValidateForQuestion(q.Type, q.Options)
}

func (o QuizOption) Validate() error {
	if err := validatePositiveInt("quiz option order", o.Order); err != nil {
		return err
	}
	return requireNotBlank("quiz option text", o.Text)
}

func (a QuizAnswer) ValidateForQuestion(questionType QuizQuestionType, options []QuizOption) error {
	switch questionType {
	case QuizQuestionTypeSingleChoice:
		if len(options) < 2 {
			return fmt.Errorf("%w: single choice question must have at least two options", ErrInvalidCollection)
		}
		return validateSingleAnswer(a, options, "single choice answer")
	case QuizQuestionTypeTrueFalse:
		if len(options) != 2 {
			return fmt.Errorf("%w: true false question must have exactly two options", ErrInvalidCollection)
		}
		return validateSingleAnswer(a, options, "true false answer")
	case QuizQuestionTypeShortAnswer:
		if len(options) > 0 {
			return fmt.Errorf("%w: short answer question cannot have options", ErrInvalidCollection)
		}
		if len(normalizeStringSlice(a.Answers)) > 0 {
			return fmt.Errorf("%w: short answer must use answer, not answers", ErrInvalidQuizAnswer)
		}
		return requireNotBlank("short answer", optionalAnswerValue(a.Answer))
	case QuizQuestionTypeMultipleChoice:
		if a.Answer != nil && strings.TrimSpace(*a.Answer) != "" {
			return fmt.Errorf("%w: multiple choice must use answers, not answer", ErrInvalidQuizAnswer)
		}
		answers := normalizeStringSlice(a.Answers)
		if len(answers) == 0 {
			return fmt.Errorf("%w: multiple choice answers", ErrInvalidQuizAnswer)
		}
		if len(options) < 3 {
			return fmt.Errorf("%w: multiple choice question must have at least three options", ErrInvalidCollection)
		}
		if err := validateUniqueQuizAnswers(answers); err != nil {
			return err
		}
		for _, answer := range answers {
			if !quizAnswerMatchesOption(answer, options) {
				return fmt.Errorf("%w: answer does not match an option", ErrInvalidQuizAnswer)
			}
		}
		return nil
	default:
		return questionType.Validate()
	}
}

func validateSingleAnswer(answer QuizAnswer, options []QuizOption, fieldName string) error {
	if len(normalizeStringSlice(answer.Answers)) > 0 {
		return fmt.Errorf("%w: %s must use answer, not answers", ErrInvalidQuizAnswer, fieldName)
	}
	value := optionalAnswerValue(answer.Answer)
	if err := requireNotBlank(fieldName, value); err != nil {
		return err
	}
	if len(options) == 0 {
		return fmt.Errorf("%w: %s options", ErrInvalidCollection, fieldName)
	}
	if !quizAnswerMatchesOption(value, options) {
		return fmt.Errorf("%w: answer does not match an option", ErrInvalidQuizAnswer)
	}
	return nil
}

func optionalAnswerValue(value *string) string {
	if value == nil {
		return ""
	}
	return normalizeText(*value)
}

func quizAnswerMatchesOption(answer string, options []QuizOption) bool {
	normalizedAnswer := normalizeText(answer)
	for _, option := range options {
		if strings.EqualFold(normalizedAnswer, normalizeText(option.Text)) {
			return true
		}
	}
	return false
}

func normalizeQuizQuestions(questions []QuizQuestion) []QuizQuestion {
	normalized := make([]QuizQuestion, 0, len(questions))
	for _, question := range questions {
		normalized = append(normalized, normalizeQuizQuestion(question))
	}
	return normalized
}

func normalizeQuizQuestion(question QuizQuestion) QuizQuestion {
	question.Question = normalizeText(question.Question)
	question.Correction = normalizeMarkdown(question.Correction)
	question.Options = normalizeQuizOptions(question.Options)
	question.Answer = normalizeQuizAnswer(question.Answer)
	return question
}

func normalizeQuizOptions(options []QuizOption) []QuizOption {
	normalized := make([]QuizOption, 0, len(options))
	for _, option := range options {
		option.Text = normalizeText(option.Text)
		normalized = append(normalized, option)
	}
	return normalized
}

func normalizeQuizAnswer(answer QuizAnswer) QuizAnswer {
	answer.Answer = trimOptionalString(answer.Answer)
	answer.Answers = normalizeStringSlice(answer.Answers)
	return answer
}

func validateContiguousQuizQuestionOrders(questions []QuizQuestion) error {
	seen := make(map[int]struct{}, len(questions))
	for _, question := range questions {
		if question.Order <= 0 {
			return fmt.Errorf("%w: quiz question order", ErrInvalidOrder)
		}
		if _, ok := seen[question.Order]; ok {
			return ErrDuplicateQuestionOrder
		}
		seen[question.Order] = struct{}{}
	}
	for order := 1; order <= len(questions); order++ {
		if _, ok := seen[order]; !ok {
			return fmt.Errorf("%w: quiz question orders must start at 1 and be contiguous", ErrInvalidCollection)
		}
	}
	return nil
}

func validateContiguousQuizOptionOrders(options []QuizOption) error {
	seen := make(map[int]struct{}, len(options))
	for _, option := range options {
		if option.Order <= 0 {
			return fmt.Errorf("%w: quiz option order", ErrInvalidOrder)
		}
		if _, ok := seen[option.Order]; ok {
			return ErrDuplicateOptionOrder
		}
		seen[option.Order] = struct{}{}
	}
	for order := 1; order <= len(options); order++ {
		if _, ok := seen[order]; !ok {
			return fmt.Errorf("%w: quiz option orders must start at 1 and be contiguous", ErrInvalidCollection)
		}
	}
	return nil
}

func validateUniqueQuizAnswers(answers []string) error {
	seen := make(map[string]struct{}, len(answers))
	for _, answer := range answers {
		normalizedAnswer := strings.ToLower(normalizeText(answer))
		if normalizedAnswer == "" {
			continue
		}
		if _, ok := seen[normalizedAnswer]; ok {
			return fmt.Errorf("%w: duplicate multiple choice answer", ErrInvalidQuizAnswer)
		}
		seen[normalizedAnswer] = struct{}{}
	}
	return nil
}
