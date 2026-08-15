import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { renderHook } from "@testing-library/react";
import { type ReactNode } from "react";
import { describe, expect, it, vi } from "vitest";
import { getRuntimeConfig } from "../../../lib/runtime-config";
import { getHomeRuntime } from "../../home/state/home-runtime";
import { useAccountSettings } from "./use-account-settings";

const mocks = vi.hoisted(() => ({
  requestLogout: vi.fn(),
  requestSession: vi.fn(),
  requestMe: vi.fn(),
  requestDeleteAccount: vi.fn(),
  requestGoogleStart: vi.fn(),
  requestDiscordStart: vi.fn(),
  requestUnlinkAuthProvider: vi.fn(),
  routerPush: vi.fn(),
}));

vi.mock("../../auth/lib/auth-client", () => mocks);
vi.mock("next/router", () => ({
  useRouter: () => ({ push: mocks.routerPush }),
}));

function renderAccountSettings() {
  const client = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={client}>{children}</QueryClientProvider>
  );
  return renderHook(() => useAccountSettings("/players/test"), { wrapper });
}

describe("useAccountSettings", () => {
  it("clears the shared session before navigating home after sign out", async () => {
    mocks.requestSession.mockResolvedValue(null);
    mocks.requestLogout.mockResolvedValue(undefined);
    const { result } = renderAccountSettings();

    await result.current.signOut();

    const sessionController = getHomeRuntime(getRuntimeConfig()).sessionController;
    expect(mocks.requestLogout).toHaveBeenCalled();
    expect(sessionController.getState().userId).toBe("");
    expect(sessionController.getState().userEmail).toBe("");
    expect(sessionController.getState().accessToken).toBe("");
    expect(mocks.routerPush).toHaveBeenCalledWith("/");
  });

  it("clears the shared session after the account is deleted", async () => {
    mocks.requestSession.mockResolvedValue(null);
    mocks.requestDeleteAccount.mockResolvedValue(undefined);
    const { result } = renderAccountSettings();

    result.current.deleteMutation.mutate();

    await vi.waitFor(() => expect(mocks.routerPush).toHaveBeenCalledWith("/"));
    const sessionController = getHomeRuntime(getRuntimeConfig()).sessionController;
    expect(sessionController.getState().userId).toBe("");
  });
});
