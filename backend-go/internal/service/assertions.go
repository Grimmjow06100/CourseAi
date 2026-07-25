package service

import "github.com/Grimmjow06100/course-ai/backend-go/internal/contract"

var _ contract.AuthService = (*AuthService)(nil)
var _ contract.CourseCatalogService = (*CourseCatalogService)(nil)
var _ contract.CourseGenerationService = (*CourseGeneratorService)(nil)
