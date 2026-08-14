import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { FlagSelect } from "./FlagSelect";

afterEach(cleanup);

describe("FlagSelect", () => {
  it("renders a plus trigger when no flag is selected", () => {
    render(<FlagSelect value="" onSelect={() => {}} />);
    const trigger = screen.getByLabelText("Select national flag");
    expect(trigger.tagName).toBe("BUTTON");
    expect(trigger.querySelector("svg")).not.toBeNull();
  });

  it("renders the selected flag image instead of the plus icon", () => {
    render(<FlagSelect value="US" onSelect={() => {}} />);
    const trigger = screen.getByLabelText("National flag: US. Change flag.");
    const img = trigger.querySelector("img");
    expect(img).not.toBeNull();
    expect(img?.getAttribute("src")).toBe("https://flagcdn.com/us.svg");
  });

  it("opens a visual menu and auto-saves the chosen flag on click", () => {
    const onSelect = vi.fn();
    render(<FlagSelect value="" onSelect={onSelect} />);
    fireEvent.click(screen.getByLabelText("Select national flag"));

    const menu = screen.getByRole("menu");
    expect(menu).toBeInTheDocument();

    fireEvent.click(screen.getByLabelText("Flag United States"));
    expect(onSelect).toHaveBeenCalledWith("US");
    expect(screen.queryByRole("menu")).not.toBeInTheDocument();
  });

  it("filters by country name as well as by code", () => {
    const onSelect = vi.fn();
    render(<FlagSelect value="" onSelect={onSelect} />);
    fireEvent.click(screen.getByLabelText("Select national flag"));

    const search = screen.getByLabelText("Search flags");
    fireEvent.change(search, { target: { value: "germany" } });
    expect(screen.getByLabelText("Flag Germany")).toBeInTheDocument();
    expect(screen.queryByLabelText("Flag United States")).not.toBeInTheDocument();

    fireEvent.change(search, { target: { value: "de" } });
    expect(screen.getByLabelText("Flag Germany")).toBeInTheDocument();
  });

  it("clears the selection via the None action", () => {
    const onSelect = vi.fn();
    render(<FlagSelect value="US" onSelect={onSelect} />);
    fireEvent.click(screen.getByLabelText("National flag: US. Change flag."));
    fireEvent.click(screen.getByText("None"));
    expect(onSelect).toHaveBeenCalledWith("");
  });

  it("disables the trigger while saving", () => {
    render(<FlagSelect value="" onSelect={() => {}} isSaving />);
    expect(screen.getByLabelText("Select national flag")).toBeDisabled();
  });
});
