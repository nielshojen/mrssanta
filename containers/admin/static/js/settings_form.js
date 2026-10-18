// Fill and read the preflight settings fields rendered by
// templates/_settings_form.html. An empty field means "not set".

function fillSettingsForm(container, settings) {
    settings = settings || {};
    if (window.closeRegexBuilders) closeRegexBuilders(container);

    container.querySelectorAll("[data-setting]").forEach(function(field) {
        const key = field.dataset.setting;
        const value = settings[key];

        if (field.dataset.type === "media") {
            field.value = value && value.action ? value.action : "";
            const flags = container.querySelector(`[data-setting-flags="${key}"]`);
            if (flags) flags.value = value && value.remount_flags ? value.remount_flags.join(",") : "";
        } else if (value === undefined || value === null) {
            field.value = "";
        } else {
            field.value = String(value);
        }
    });
}

function readSettingsForm(container) {
    const settings = {};

    container.querySelectorAll("[data-setting]").forEach(function(field) {
        const key = field.dataset.setting;

        if (field.dataset.type === "media") {
            const flags = container.querySelector(`[data-setting-flags="${key}"]`);
            settings[key] = field.value
                ? {action: field.value, remount_flags: flags ? flags.value : ""}
                : null;
        } else {
            settings[key] = field.value.trim();
        }
    });

    return settings;
}
