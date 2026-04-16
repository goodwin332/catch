"use client";

import { useEffect, useState } from "react";

export function ThemeToggle() {
  const [dark, setDark] = useState(false);

  useEffect(() => {
    const saved = window.localStorage.getItem("catch-theme");
    const nextDark = saved === "dark";
    setDark(nextDark);
    document.documentElement.dataset.theme = nextDark ? "dark" : "light";
  }, []);

  function toggle() {
    const nextDark = !dark;
    setDark(nextDark);
    document.documentElement.dataset.theme = nextDark ? "dark" : "light";
    window.localStorage.setItem("catch-theme", nextDark ? "dark" : "light");
  }

  return (
    <button className="theme-toggle" type="button" onClick={toggle} aria-label="Переключить тему">
      {dark ? "Светлая" : "Тёмная"}
    </button>
  );
}
