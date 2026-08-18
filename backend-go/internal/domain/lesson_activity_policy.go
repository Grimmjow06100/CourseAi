package domain

import "fmt"

func ValidateLessonActivitiesForType(lessonType LessonType, exercises []Exercise, quizzes []Quiz) error {
	if err := lessonType.Validate(); err != nil {
		return err
	}

	switch lessonType {
	case LessonTypePractice:
		if len(exercises) == 0 {
			return fmt.Errorf("%w: practice lesson must include at least one exercise", ErrInvalidCollection)
		}
		if len(quizzes) > 0 {
			return fmt.Errorf("%w: practice lesson must not include quizzes", ErrInvalidCollection)
		}
	case LessonTypeQuiz:
		if len(quizzes) == 0 {
			return fmt.Errorf("%w: quiz lesson must include at least one quiz", ErrInvalidCollection)
		}
		if len(exercises) > 0 {
			return fmt.Errorf("%w: quiz lesson must not include exercises", ErrInvalidCollection)
		}
	case LessonTypeMixed:
		if len(exercises) == 0 || len(quizzes) == 0 {
			return fmt.Errorf("%w: mixed lesson must include at least one exercise and one quiz", ErrInvalidCollection)
		}
	case LessonTypeTheory:
		if len(exercises) > 0 {
			return fmt.Errorf("%w: theory lesson must not include exercises", ErrInvalidCollection)
		}
	}

	return nil
}
