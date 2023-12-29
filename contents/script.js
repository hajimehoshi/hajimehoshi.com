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
