import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { useState, type FormEvent } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";

type Status = "idle" | "invalid" | "saving" | "saved";

function CityForm({ onSave }: { onSave: (city: string) => Promise<void> }) {
  const [city, setCity] = useState("");
  const [status, setStatus] = useState<Status>("idle");
  async function submit(e: FormEvent) {
    e.preventDefault();
    if (!city.trim()) return setStatus("invalid");
    setStatus("saving");
    await onSave(city.trim().toLocaleUpperCase("tr"));
    setStatus("saved");
  }
  return (
    <form onSubmit={submit}>
      <label>
        City <input value={city} onChange={(e) => setCity(e.target.value)} />
      </label>
      <button disabled={status === "saving"}>Save</button>
      {status === "invalid" && <p role="alert">Enter a city</p>}
      {status === "saved" && <p role="status">Saved</p>}
    </form>
  );
}

afterEach(cleanup); // unmount between tests (see the annotation)

describe("CityForm", () => {
  it("asks for a city instead of saving an empty one", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    render(<CityForm onSave={onSave} />);
    await user.click(screen.getByRole("button", { name: "Save" }));
    expect(screen.getByRole("alert").textContent).toBe("Enter a city");
    expect(onSave).not.toHaveBeenCalled();
  });

  it("disables Save while saving, then confirms", async () => {
    const user = userEvent.setup();
    const reply = Promise.withResolvers<void>();
    const onSave = vi.fn(() => reply.promise);
    render(<CityForm onSave={onSave} />);
    const button = screen.getByRole("button", { name: "Save" });
    await user.type(screen.getByLabelText("City"), "  izmir ");
    await user.click(button);
    expect(button).toHaveProperty("disabled", true);
    expect(onSave).toHaveBeenCalledExactlyOnceWith("İZMİR");
    reply.resolve(); // the "server" answers
    expect((await screen.findByRole("status")).textContent).toBe("Saved");
    expect(screen.queryByRole("alert")).toBeNull();
  });
});
