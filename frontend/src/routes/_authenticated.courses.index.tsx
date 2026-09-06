import { getErrorMessage } from '@/shared/api/error-message'
import { createFileRoute } from '@tanstack/react-router'
import { Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCourses } from '@/features/catalog/api'
import { CourseCard } from '@/features/catalog/course-card'
import { catalogSearchSchema } from '@/features/catalog/schemas'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/feedback'
import { DebouncedSearch } from '@/shared/ui/debounced-search'
import { Select } from '@/shared/ui/form-controls'
import { PageHeader, Pagination } from '@/shared/ui/page'

function CatalogPage() {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const query = useCourses(search)
  return (
    <div>
      <PageHeader title={t('catalog.title')} description={t('catalog.subtitle')} />
      <div className="mb-7 grid gap-3 border-y border-border py-4 md:grid-cols-2 xl:grid-cols-[minmax(240px,1fr)_180px_210px_180px]">
        <label className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <DebouncedSearch
            navigationKey={JSON.stringify(search)}
            initialValue={search.search ?? ''}
            label={t('catalog.search')}
            onSearch={(value) =>
              void navigate({
                search: (previous) => ({ ...previous, search: value || undefined, page: 1 }),
                replace: true,
              })
            }
          />
        </label>
        <Select
          aria-label={t('catalog.allLanguages')}
          value={search.language ?? ''}
          onChange={(event) =>
            void navigate({
              search: (previous) => ({
                ...previous,
                language: (event.target.value || undefined) as 'fr' | 'en' | undefined,
                page: 1,
              }),
            })
          }
        >
          <option value="">{t('catalog.allLanguages')}</option>
          <option value="fr">Français</option>
          <option value="en">English</option>
        </Select>
        <Select
          aria-label={t('catalog.allStatuses')}
          value={search.status ?? ''}
          onChange={(event) =>
            void navigate({
              search: (previous) => ({
                ...previous,
                status: (event.target.value || undefined) as typeof search.status,
                page: 1,
              }),
            })
          }
        >
          <option value="">{t('catalog.allStatuses')}</option>
          <option value="completed">{t('course.status.completed')}</option>
          <option value="lessons_generated">{t('course.status.lessons_generated')}</option>
          <option value="content_generating">{t('course.status.content_generating')}</option>
          <option value="failed">{t('course.status.failed')}</option>
        </Select>
        <Select
          aria-label={t('catalog.sortNewest')}
          value={search.orderBy}
          onChange={(event) =>
            void navigate({
              search: (previous) => ({
                ...previous,
                orderBy: event.target.value as typeof search.orderBy,
                orderDirection: event.target.value === 'title' ? 'asc' : 'desc',
                page: 1,
              }),
            })
          }
        >
          <option value="created_at">{t('catalog.sortNewest')}</option>
          <option value="updated_at">{t('catalog.sortUpdated')}</option>
          <option value="title">{t('catalog.sortTitle')}</option>
        </Select>
      </div>
      {query.isLoading ? (
        <LoadingState label={t('common.loading')} />
      ) : query.error ? (
        <ErrorState
          title={t('errors.generic')}
          description={getErrorMessage(query.error, t)}
          onRetry={() => void query.refetch()}
        />
      ) : query.data?.items.length ? (
        <>
          <div className="grid gap-5 md:grid-cols-2 2xl:grid-cols-3">
            {query.data.items.map((course) => (
              <CourseCard key={course.id} course={course} />
            ))}
          </div>
          <Pagination
            page={query.data.page}
            totalPages={query.data.totalPages}
            hasPrevious={query.data.hasPrevious}
            hasNext={query.data.hasNext}
            label={t('common.page', { current: query.data.page, total: query.data.totalPages })}
            onPageChange={(page) => void navigate({ search: (previous) => ({ ...previous, page }) })}
          />
        </>
      ) : (
        <EmptyState title={t('catalog.empty')} />
      )}
    </div>
  )
}
export const Route = createFileRoute('/_authenticated/courses/')({
  validateSearch: catalogSearchSchema,
  component: CatalogPage,
})
