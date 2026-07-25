package openai

func analysisSchema() map[string]any {
	return objectSchema(
		[]string{
			"isOutOfScope",
			"errorMessage",
			"warningMessage",
			"suggestedTitle",
			"shortSynopsis",
			"detectedCurrentLevel",
			"detectedTargetLevel",
			"detectedGoal",
			"detectedLanguage",
			"clarificationQuestions",
		},
		map[string]any{
			"isOutOfScope":         boolSchema(),
			"errorMessage":         nullableStringSchema(),
			"warningMessage":       nullableStringSchema(),
			"suggestedTitle":       stringSchema(),
			"shortSynopsis":        stringSchema(),
			"detectedCurrentLevel": enumSchema([]string{"beginner", "intermediate", "advanced", "unknow", "unknown"}),
			"detectedTargetLevel":  enumSchema([]string{"beginner", "intermediate", "advanced", "expert", "unknow", "unknown"}),
			"detectedGoal":         stringSchema(),
			"detectedLanguage":     enumSchema([]string{"fr", "en"}),
			"clarificationQuestions": arraySchema(objectSchema(
				[]string{"id", "question", "options"},
				map[string]any{
					"id":       enumSchema([]string{"goals", "currentLevel", "targetLevel"}),
					"question": stringSchema(),
					"options":  arraySchema(stringSchema()),
				},
			)),
		},
	)
}

func architectureSchema() map[string]any {
	moduleSchema := objectSchema(
		[]string{"order", "title", "description", "keyLearningPoints"},
		map[string]any{
			"order":             integerSchema(),
			"title":             stringSchema(),
			"description":       stringSchema(),
			"keyLearningPoints": arraySchema(stringSchema()),
		},
	)

	finalProjectSchema := objectSchema(
		[]string{"title", "description", "constraints"},
		map[string]any{
			"title":       stringSchema(),
			"description": stringSchema(),
			"constraints": arraySchema(stringSchema()),
		},
	)

	return objectSchema(
		[]string{"title", "synopsis", "targetAudience", "prerequisites", "goals", "acquiredSkills", "modules", "finalProject"},
		map[string]any{
			"title":          stringSchema(),
			"synopsis":       stringSchema(),
			"targetAudience": stringSchema(),
			"prerequisites":  arraySchema(stringSchema()),
			"goals":          arraySchema(stringSchema()),
			"acquiredSkills": arraySchema(stringSchema()),
			"modules":        arraySchema(moduleSchema),
			"finalProject":   finalProjectSchema,
		},
	)
}

func lessonsSchema() map[string]any {
	lessonSchema := objectSchema(
		[]string{"order", "title", "type", "estimatedDuration", "learningGoal", "requiresDiagram", "technicalKeywords"},
		map[string]any{
			"order":             integerSchema(),
			"title":             stringSchema(),
			"type":              enumSchema([]string{"theory", "practice", "mixed", "quiz"}),
			"estimatedDuration": integerSchema(),
			"learningGoal":      stringSchema(),
			"requiresDiagram":   boolSchema(),
			"technicalKeywords": arraySchema(stringSchema()),
		},
	)

	return objectSchema(
		[]string{"moduleOrder", "moduleTitle", "lessons"},
		map[string]any{
			"moduleOrder": integerSchema(),
			"moduleTitle": stringSchema(),
			"lessons":     arraySchema(lessonSchema),
		},
	)
}

func lessonContentSchema() map[string]any {
	return objectSchema(
		[]string{"contentMarkdown"},
		map[string]any{
			"contentMarkdown": stringSchema(),
		},
	)
}

func objectSchema(required []string, properties map[string]any) map[string]any {
	return map[string]any{
		"type":                 "object",
		"additionalProperties": false,
		"required":             required,
		"properties":           properties,
	}
}

func arraySchema(items map[string]any) map[string]any {
	return map[string]any{
		"type":  "array",
		"items": items,
	}
}

func stringSchema() map[string]any {
	return map[string]any{"type": "string"}
}

func nullableStringSchema() map[string]any {
	return map[string]any{"type": []string{"string", "null"}}
}

func boolSchema() map[string]any {
	return map[string]any{"type": "boolean"}
}

func integerSchema() map[string]any {
	return map[string]any{"type": "integer"}
}

func enumSchema(values []string) map[string]any {
	return map[string]any{
		"type": "string",
		"enum": values,
	}
}
