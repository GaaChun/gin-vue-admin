import { afterEach } from 'vitest'

afterEach(() => {
  // happy-dom shares its document between tests, so mounted Vue nodes must not leak across cases.
  document.body.replaceChildren()
})
