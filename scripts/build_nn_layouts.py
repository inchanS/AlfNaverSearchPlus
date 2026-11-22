"""Build ready-to-open Alfred UI packages for the Configure dropdowns."""

import json
import shutil
from itertools import product
from pathlib import Path


def expand_files(value, source):
    # Inline static includes at build time. Alfred still evaluates variables and
    # component values at their usual points in the UI and its actions.
    if isinstance(value, str) and value.startswith("{file:") and value.endswith("}"):
        return expand_files(json.loads((source / value[6:-1]).read_text()), source)
    if isinstance(value, list):
        return [expand_files(item, source) for item in value]
    if isinstance(value, dict):
        return {key: expand_files(item, source) for key, item in value.items()}
    return value


def build_layouts(workflow):
    source = workflow / "nui.alfredui"
    ui = expand_files(json.loads((source / "ui.json").read_text()), source)
    default = ui["variables"].copy()
    default_size = ui["window"]["size"]
    fonts = {
        "default": (default["nn_primary_font"], default["nn_query_font"],
                    default["nn_secondary_font"], default["nn_service_font"],
                    default["nn_row_height"], default_size["height"] - default["nn_list_height"]),
        "large": ("large", "extralarge", "default", "default", 51, 191),
        "extra_large": ("extralarge", "ginormous", "large", "medium", 59, 207),
    }
    widths = {"theme": None, "current": default_size["width"], "wide": 760}
    for width, font, rows in product(widths, fonts, (6, 8, 10)):
        primary, query, secondary, service, row_height, chrome = fonts[font]
        list_height = rows * row_height
        ui["variables"].update(
            nn_primary_font=primary, nn_query_font=query,
            nn_secondary_font=secondary, nn_service_font=service,
            nn_row_height=row_height, nn_list_height=list_height,
        )
        size = {"height": list_height + chrome}
        if widths[width] is not None:
            size["width"] = widths[width]
        ui["window"]["size"] = size

        package = workflow / f"nui-{width}-{font}-{rows}.alfredui"
        package.mkdir(exist_ok=True)
        shutil.copy2(source / "run", package / "run")
        (package / "ui.json").write_text(json.dumps(ui, ensure_ascii=False) + "\n")


if __name__ == "__main__":
    build_layouts(Path(__file__).resolve().parents[1] / "workflow")
