import { generationKeys } from './query-keys'

it('includes every list filter in generation query keys', () => {
  expect(generationKeys.list({ status: 'running', page: 2, pageSize: 20 })).toEqual([
    'generations',
    'list',
    { status: 'running', page: 2, pageSize: 20 },
  ])
})
