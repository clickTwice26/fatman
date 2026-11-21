"use client";

import { createTheme } from "@mui/material/styles";

export const FONT_FAMILY_ONEST = "var(--font-onest), 'Onest', system-ui, -apple-system, sans-serif";

export const fatmanTheme = createTheme({
  cssVariables: {
    colorSchemeSelector: "media",
  },
  colorSchemes: {
    light: {
      palette: {
        primary: {
          main: "#147A58",
          light: "#2FA17A",
          dark: "#0C573E",
          contrastText: "#FFFFFF",
        },
        secondary: {
          main: "#E06D3B",
          light: "#F19066",
          dark: "#B54D21",
          contrastText: "#FFFFFF",
        },
        background: {
          default: "#F7F9F7",
          paper: "#FFFFFF",
        },
        text: {
          primary: "#13221C",
          secondary: "#4A6158",
        },
        divider: "rgba(19, 34, 28, 0.1)",
      },
    },
    dark: {
      palette: {
        primary: {
          main: "#4AD29E",
          light: "#7CE3BA",
          dark: "#26A476",
          contrastText: "#062117",
        },
        secondary: {
          main: "#F49367",
          light: "#F9B596",
          dark: "#D66B3B",
          contrastText: "#230E05",
        },
        background: {
          default: "#0D1512",
          paper: "#14201B",
        },
        text: {
          primary: "#EDF5F1",
          secondary: "#A5BCB2",
        },
        divider: "rgba(237, 245, 241, 0.12)",
      },
    },
  },
  shape: {
    borderRadius: 14,
  },
  typography: {
    fontFamily: FONT_FAMILY_ONEST,
    h1: { fontWeight: 700, letterSpacing: "-0.025em" },
    h2: { fontWeight: 700, letterSpacing: "-0.02em" },
    h3: { fontWeight: 600, letterSpacing: "-0.015em" },
    h4: { fontWeight: 600 },
    h5: { fontWeight: 600 },
    h6: { fontWeight: 600 },
    button: { fontWeight: 600, textTransform: "none" },
  },
  components: {
    MuiButton: {
      defaultProps: {
        disableElevation: true,
      },
      styleOverrides: {
        root: {
          borderRadius: 12,
          paddingInline: 20,
          paddingBlock: 10,
        },
      },
    },
    MuiCard: {
      defaultProps: {
        variant: "outlined",
      },
      styleOverrides: {
        root: {
          borderRadius: 18,
        },
      },
    },
  },
});
