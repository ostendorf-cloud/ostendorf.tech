import { onBeforeUnmount, onMounted, ref, type Ref } from 'vue'

/**
 * useMediaQuery follows a CSS media query for the lifetime of the component.
 * It is false on the server, before mount and wherever matchMedia does not
 * exist, so the first client render agrees with the prerendered one.
 */
export function useMediaQuery(query: string): Ref<boolean> {
  const matches = ref(false)
  let list: MediaQueryList | undefined
  const update = (): void => {
    matches.value = list?.matches ?? false
  }
  onMounted(() => {
    if (typeof window === 'undefined' || typeof window.matchMedia !== 'function') {
      return
    }
    list = window.matchMedia(query)
    update()
    list.addEventListener('change', update)
  })
  onBeforeUnmount(() => {
    list?.removeEventListener('change', update)
  })
  return matches
}
