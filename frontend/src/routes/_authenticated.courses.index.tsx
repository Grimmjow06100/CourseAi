import { getErrorMessage } from '@/shared/api/error-message'
import { createFileRoute, Link } from '@tanstack/react-router'
import { Plus, Search } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useCourses } from '@/features/catalog/api'
import { CourseCard } from '@/features/catalog/course-card'
import { catalogSearchSchema } from '@/features/catalog/schemas'
import { EmptyState, ErrorState, LoadingState } from '@/shared/ui/feedback'
import { DebouncedSearch } from '@/shared/ui/debounced-search'
import { Select } from '@/shared/ui/form-controls'
import { PageHeader, Pagination } from '@/shared/ui/page'
import { Button } from '@/shared/ui/button'

function CatalogPage() {
  const { t } = useTranslation()
  const search = Route.useSearch()
  const navigate = Route.useNavigate()
  const query = useCourses(search)
  const filtered = Boolean(search.search) || Boolean(search.language) || Boolean(search.status)
  const resetFilters = () =>
    void navigate({ search: { page: 1, orderBy: search.orderBy, orderDirection: search.orderDirection } })
  return (
    <div>
      <PageHeader
        eyebrow={t('design.workspace')}
        title={t('catalog.title')}
        description={t('catalog.subtitle')}
        actions={
          <Button asChild>
            <Link to="/generate">
              <Plus className="size-4" />
              {t('nav.generate')}
            </Link>
          </Button>
        }
      />
      <div className="studio-panel mb-6 p-4">
        <div className="mb-3 flex min-h-9 items-center justify-between gap-3">
          <h2 className="text-xs font-bold text-muted-foreground">{t('design.filters')}</h2>
          {filtered ? (
            <Button variant="ghost" size="sm" onClick={resetFilters}>
              {t('design.reset')}
            </Button>
          ) : null}
        </div>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-[minmax(180px,1fr)_160px_190px_170px]">
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
            {catalogSearchSchema.shape.status
              .unwrap()
              .unwrap()
              .options.map((status) => (
                <option key={status} value={status}>
                  {t(`course.status.${status}`)}
                </option>
              ))}
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
            <option value="status">{t('catalog.allStatuses')}</option>
          </Select>
        </div>
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
          <p className="mb-4 text-xs font-semibold text-muted-foreground" role="status">
            {t('design.results', { count: query.data.totalItems })}
          </p>
          <div className="grid gap-5 md:grid-cols-2 min-[1380px]:grid-cols-3">
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
        <EmptyState
          title={filtered ? t('catalog.empty') : t('dashboard.emptyCourses')}
          description={filtered ? t('design.filteredEmpty') : t('design.emptyCoursesText')}
          action={
            filtered ? (
              <Button variant="secondary" onClick={resetFilters}>
                {t('design.reset')}
              </Button>
            ) : (
              <Button asChild>
                <Link to="/generate">
                  <Plus className="size-4" />
                  {t('nav.generate')}
                </Link>
              </Button>
            )
          }
        />
      )}
    </div>
  )
}
export const Route = createFileRoute('/_authenticated/courses/')({
  validateSearch: catalogSearchSchema,
  component: CatalogPage,
})
