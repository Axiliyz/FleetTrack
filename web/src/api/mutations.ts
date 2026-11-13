import { useMutation, useQueryClient, type QueryKey } from '@tanstack/react-query'

// useApiMutation — изменение, после которого перечитываются затронутые списки.
// Ответ мутации в кеш не подкладываем: списки отфильтрованы и постраничны, проще и честнее перечитать.
// Перечитываем и после ошибки: составная операция могла упасть на втором запросе, когда первый уже прошёл
export function useApiMutation<TVars, TData = unknown>(fn: (vars: TVars) => Promise<TData>, invalidate: QueryKey[]) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: fn,
    onSettled: () => Promise.all(invalidate.map((queryKey) => queryClient.invalidateQueries({ queryKey }))),
  })
}
