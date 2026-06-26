addEventListener("DOMContentLoaded", (e) => {
    const darkModeCheckbox = document.getElementById("dark-mode");

    darkModeCheckbox.addEventListener('change', (e) => {
        localStorage.setItem('darkMode', JSON.stringify(e.target.checked));
    });

    let darkMode = false;
    if (localStorage.getItem('darkMode') !== null) {
        try {
            darkMode = !!JSON.parse(localStorage.getItem('darkMode'));
        } catch (e) {
            console.error(e);
        }
    } else if (window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches) {
        darkMode = true;
    }
    darkModeCheckbox.checked = darkMode;
});

// Snap each content block's height up to the 168px grid. (In JS because CSS
// can't read an element's intrinsic height back into calc().)
function snapModularHeights() {
    const UNIT = 24;
    const MODULE = 7 * UNIT;    // 168px column module
    const STEP = MODULE + UNIT; // 192px: one module plus one gap

    const blocks = document.querySelectorAll(
        ".grid-container, section > header, section > ul, section > dl");

    // Reset first so each block is measured at its natural height.
    for (const block of blocks) {
        block.style.minHeight = "";
    }
    for (const block of blocks) {
        // Round away sub-pixel measurement noise, then snap up to the grid.
        const height = Math.round(block.getBoundingClientRect().height);
        const n = Math.max(1, Math.ceil((height + UNIT) / STEP));
        block.style.minHeight = `${STEP * n - UNIT}px`;
        block.style.alignContent = "start";
    }
}

addEventListener("load", snapModularHeights);
addEventListener("resize", snapModularHeights);
if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(snapModularHeights);
}
