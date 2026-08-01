// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2026 Hajime Hoshi

const hhdcDarkModeStorageKey = "hhdcDarkMode";

try {
  const darkMode = localStorage.getItem(hhdcDarkModeStorageKey);
  if (darkMode === "true") {
    document.documentElement.dataset.hhdcColorMode = "dark";
  } else if (darkMode === "false") {
    document.documentElement.dataset.hhdcColorMode = "light";
  }
} catch (e) {
  console.error(e);
}

const hhdcColorModeToggleTemplate = document.createElement("template");
hhdcColorModeToggleTemplate.innerHTML = `
  <style>
    :host {
      display: flex;
      width: 1.5rem;
      height: 1.5rem;
    }
    label {
      display: flex;
      align-items: center;
      width: 100%;
      height: 100%;
      margin: -25%;
      padding: 25%;
      cursor: pointer;
      user-select: none;
    }
    input {
      position: absolute;
      width: 1px;
      height: 1px;
      overflow: hidden;
      clip-path: inset(50%);
      white-space: nowrap;
    }
    .icons {
      display: flex;
      align-items: center;
      width: 100%;
      height: 100%;
    }
    input:focus-visible + .icons {
      outline: 2px solid currentColor;
      outline-offset: 2px;
      border-radius: calc(100% / 6);
    }
    .light {
      display: none;
    }
    input:checked + .icons .light {
      display: block;
    }
    input:checked + .icons .dark {
      display: none;
    }
    svg {
      width: 100%;
      height: 100%;
    }
    path {
      fill: currentColor;
    }
  </style>
  <label>
    <input type="checkbox" role="switch" aria-label="Dark mode">
    <span class="icons">
      <!-- https://fonts.google.com/icons light-mode -->
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" class="light" aria-hidden="true"><path d="M480-360q50 0 85-35t35-85q0-50-35-85t-85-35q-50 0-85 35t-35 85q0 50 35 85t85 35Zm0 80q-83 0-141.5-58.5T280-480q0-83 58.5-141.5T480-680q83 0 141.5 58.5T680-480q0 83-58.5 141.5T480-280ZM200-440H40v-80h160v80Zm720 0H760v-80h160v80ZM440-760v-160h80v160h-80Zm0 720v-160h80v160h-80ZM256-650l-101-97 57-59 96 100-52 56Zm492 496-97-101 53-55 101 97-57 59Zm-98-550 97-101 59 57-100 96-56-52ZM154-212l101-97 55 53-97 101-59-57Zm326-268Z"/></svg>
      <!-- https://fonts.google.com/icons dark-mode -->
      <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 -960 960 960" class="dark" aria-hidden="true"><path d="M480-120q-150 0-255-105T120-480q0-150 105-255t255-105q14 0 27.5 1t26.5 3q-41 29-65.5 75.5T444-660q0 90 63 153t153 63q55 0 101-24.5t75-65.5q2 13 3 26.5t1 27.5q0 150-105 255T480-120Zm0-80q88 0 158-48.5T740-375q-20 5-40 8t-40 3q-123 0-209.5-86.5T364-660q0-20 3-40t8-40q-78 32-126.5 102T200-480q0 116 82 198t198 82Zm-10-270Z"/></svg>
    </span>
  </label>`;

class HHDCColorModeToggle extends HTMLElement {
  #darkModeQuery = matchMedia("(prefers-color-scheme: dark)");
  #input;
  #rootObserver;

  constructor() {
    super();

    const shadowRoot = this.attachShadow({ mode: "closed" });
    shadowRoot.append(hhdcColorModeToggleTemplate.content.cloneNode(true));
    this.#input = shadowRoot.querySelector("input");
    this.#input.addEventListener("change", this.#changeColorMode);
  }

  connectedCallback() {
    this.#darkModeQuery.addEventListener("change", this.#sync);
    this.#rootObserver = new MutationObserver(this.#sync);
    this.#rootObserver.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["data-hhdc-color-mode"],
    });
    this.#sync();
  }

  disconnectedCallback() {
    this.#darkModeQuery.removeEventListener("change", this.#sync);
    this.#rootObserver.disconnect();
  }

  #changeColorMode = () => {
    const darkMode = this.#input.checked;
    document.documentElement.dataset.hhdcColorMode = darkMode ? "dark" : "light";
    try {
      localStorage.setItem(hhdcDarkModeStorageKey, String(darkMode));
    } catch (e) {
      console.error(e);
    }
  };

  #sync = () => {
    const hhdcColorMode = document.documentElement.dataset.hhdcColorMode;
    this.#input.checked = hhdcColorMode === "dark" ||
      (!hhdcColorMode && this.#darkModeQuery.matches);
  };
}

customElements.define("hhdc-color-mode-toggle", HHDCColorModeToggle);
