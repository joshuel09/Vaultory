import { expect, test } from '@playwright/test'
import { addCollectible, gotoReady, openEditScreen, signIn } from './support'

// Feature 010: deleting is forgiving. Undo straight away, or restore from the Trash for 30 days.

async function deleteFromEditScreen(page: import('@playwright/test').Page, name: string) {
  await openEditScreen(page, name)
  await page.getByTestId('delete-collectible').click()
  await page.getByTestId('confirm-delete').click()
  await page.waitForURL(/\/collection\?.*deleted=/)
}

test.describe('the Trash', () => {
  test.beforeEach(async ({ page }) => signIn(page))

  // US1: the way back is offered where the collector lands.
  test('Undo straight after deleting puts it back', async ({ page }) => {
    const name = `Undone ${Date.now()}`
    await addCollectible(page, name)
    await deleteFromEditScreen(page, name)

    const notice = page.getByTestId('undo-delete-notice')
    await expect(notice).toBeVisible()
    await expect(page.getByText(name)).toHaveCount(0)

    await notice.getByRole('button', { name: 'Undo' }).click()
    await expect(notice).toBeHidden()
    await expect(page.getByText(name)).toBeVisible()
    // The marker is gone, so a reload does not offer to undo it again.
    await expect(page).not.toHaveURL(/deleted=/)
  })

  // US2: restore days later, from the Trash.
  test('a deleted collectible can be restored from the Trash', async ({ page }) => {
    const name = `Restored ${Date.now()}`
    await addCollectible(page, name)
    await deleteFromEditScreen(page, name)

    await gotoReady(page, '/collection/trash')
    const entry = page.getByTestId('trashed-collectible').filter({ hasText: name })
    await expect(entry).toBeVisible()
    await expect(entry).toContainText(/Removed for good on/)

    await entry.getByTestId('restore-collectible').click()
    await expect(page.getByTestId('trash-message')).toContainText('is back in your vault')

    await gotoReady(page, '/collection')
    await expect(page.getByText(name)).toBeVisible()
  })

  // US3: gone for good, behind a confirmation whose default keeps it.
  test('Delete permanently asks first, and Enter keeps it', async ({ page }) => {
    const name = `Destroyed ${Date.now()}`
    await addCollectible(page, name)
    await deleteFromEditScreen(page, name)

    await gotoReady(page, '/collection/trash')
    const entry = page.getByTestId('trashed-collectible').filter({ hasText: name })

    await entry.getByTestId('delete-permanently').click()
    const dialog = page.getByTestId('delete-permanently-dialog').filter({ hasText: name })
    await expect(dialog).toBeVisible()
    await expect(dialog).toContainText(/cannot be undone/i)
    await expect(dialog.getByTestId('cancel-delete-permanently')).toBeFocused()
    await page.keyboard.press('Enter')
    await expect(dialog).toBeHidden()
    await expect(entry).toBeVisible()

    await entry.getByTestId('delete-permanently').click()
    await dialog.getByTestId('confirm-delete-permanently').click()
    await expect(entry).toHaveCount(0)

    await gotoReady(page, '/collection/trash')
    await expect(page.getByText(name)).toHaveCount(0)
  })
})
