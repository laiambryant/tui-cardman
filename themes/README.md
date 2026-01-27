# Custom Themes

This directory contains theme files for the TUI Cardman application. Themes define the color scheme used throughout the interface.

## Default Themes

The following themes are included by default:

- `default.json` - Pink/purple theme (default)
- `dark.json` - White on black theme
- `light.json` - Blue on white theme

## Creating Custom Themes

To create a custom theme, create a new JSON file in this directory. The filename (without the `.json` extension) will be the theme name.

### Theme File Format

```json
{
  "name": "My Custom Theme",
  "focused": "205",
  "blurred": "240",
  "error": "9",
  "title": "170",
  "background": "0",
  "foreground": "15"
}
```

### Color Fields

- **name**: Display name for the theme (required)
- **focused**: Color for focused/selected elements (required)
- **blurred**: Color for unfocused/inactive elements (required)
- **error**: Color for error messages (required)
- **title**: Color for titles and headers (required)
- **background**: Background color (optional, empty string for terminal default)
- **foreground**: Foreground/text color (optional, empty string for terminal default)

### Color Values

Colors can be specified in the following formats:

1. **ANSI 256 colors** (0-255): Use the color number as a string (e.g., "205", "15")
2. **Hex colors**: Use a hex color code (e.g., "#FF00FF", "#FFFFFF")
3. **Terminal default**: Use an empty string ("") to use the terminal's default color

#### Popular ANSI Colors

| Number | Color        |
|--------|--------------|
| 0      | Black        |
| 1      | Red          |
| 2      | Green        |
| 4      | Blue         |
| 7      | Light Gray   |
| 8      | Dark Gray    |
| 9      | Bright Red   |
| 12     | Bright Blue  |
| 15     | White        |
| 57     | Dark Blue    |
| 170    | Purple       |
| 205    | Pink/Magenta |
| 229    | Yellow       |
| 240    | Gray         |

For a full list of ANSI 256 colors, see: https://en.wikipedia.org/wiki/ANSI_escape_code#8-bit

## Example Custom Theme

Here's an example of a "Solarized Dark" inspired theme:

```json
{
  "name": "Solarized Dark",
  "focused": "33",
  "blurred": "240",
  "error": "124",
  "title": "64",
  "background": "234",
  "foreground": "244"
}
```

Save this as `solarized-dark.json` and it will appear as "solarized-dark" in the theme selector.

## Using Custom Themes

1. Create your theme JSON file in the `themes/` directory
2. Open TUI Cardman and press `F1` to open Settings
3. Navigate to the "UI" tab using arrow keys or Tab
4. Select "Theme" and press Enter or use Left/Right arrows to cycle through available themes
5. Press Ctrl+S to save your changes

## Tips

- Test your theme in different terminal emulators as colors may appear differently
- Use contrast checkers to ensure readability
- Consider both light and dark terminal backgrounds if your theme doesn't set a background color
- Restart the application after creating new theme files to see them in the theme list
