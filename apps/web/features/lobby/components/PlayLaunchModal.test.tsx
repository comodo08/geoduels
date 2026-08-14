import React from 'react';
import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { PlayLaunchModal } from './PlayLaunchModal';
import type { ExtensionAvailabilityStatus } from '../../browser-extension/hooks/use-extension-availability';

afterEach(cleanup);

function renderModal(overrides: Partial<{
  extensionAvailable: boolean;
  endless: boolean;
  onEndlessChange: (value: boolean) => void;
  disabled: boolean;
}> = {}) {
  return render(
    <PlayLaunchModal
      kind="singleplayer"
      extensionAvailable={overrides.extensionAvailable ?? true}
      extensionStatus={{ state: 'ready' } as ExtensionAvailabilityStatus}
      mode="moving"
      streetNames="shown"
      endless={overrides.endless ?? false}
      onEndlessChange={overrides.onEndlessChange ?? vi.fn()}
      disabled={overrides.disabled ?? false}
      onModeChange={vi.fn()}
      onStreetNamesChange={vi.fn()}
      onClose={vi.fn()}
      onStart={vi.fn()}
    />,
  );
}

describe('PlayLaunchModal singleplayer endless toggle', () => {
  it('renders an Endless toggle for singleplayer', () => {
    renderModal();
    const toggle = screen.getByRole('switch', { name: 'Endless mode' });
    expect(toggle).toBeInTheDocument();
    expect(toggle).toHaveAttribute('aria-checked', 'false');
  });

  it('toggles endless on click', () => {
    const onEndlessChange = vi.fn();
    renderModal({ endless: false, onEndlessChange });
    const toggle = screen.getByRole('switch', { name: 'Endless mode' });
    fireEvent.click(toggle);
    expect(onEndlessChange).toHaveBeenCalledWith(true);
  });

  it('reflects the enabled state', () => {
    renderModal({ endless: true });
    const toggle = screen.getByRole('switch', { name: 'Endless mode' });
    expect(toggle).toHaveAttribute('aria-checked', 'true');
  });

  it('hides the toggle when the modal is disabled', () => {
    renderModal({ disabled: true });
    expect(screen.queryByRole('switch', { name: 'Endless mode' })).toBeNull();
  });
});
