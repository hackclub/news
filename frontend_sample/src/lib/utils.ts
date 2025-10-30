const AVAILABLE_COLORS = {
  red: "#ec3750",
  orange: "#ff8c37",
  yellow: "#f1c40f",
  green: "#33d6a6",
  cyan: "#5bc0de",
  blue: "#338eda",
  purple: "#a633d6",
};

function hexToRgb(hex: string): [number, number, number] | null {
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  return result
    ? [
        parseInt(result[1], 16),
        parseInt(result[2], 16),
        parseInt(result[3], 16),
      ]
    : null;
}

function colorDistance(color1: string, color2: string): number {
  const rgb1 = hexToRgb(color1);
  const rgb2 = hexToRgb(color2);

  if (!rgb1 || !rgb2) {
    return Infinity;
  }

  const [r1, g1, b1] = rgb1;
  const [r2, g2, b2] = rgb2;

  return Math.sqrt(
    Math.pow(r2 - r1, 2) + Math.pow(g2 - g1, 2) + Math.pow(b2 - b1, 2),
  );
}

export function getClosestColor(color: string): string {
  let closestColor = "";
  let minDistance = Infinity;

  for (const colorName in AVAILABLE_COLORS) {
    const currentColor =
      AVAILABLE_COLORS[colorName as keyof typeof AVAILABLE_COLORS];
    const distance = colorDistance(color, currentColor);

    if (distance < minDistance) {
      minDistance = distance;
      closestColor = currentColor;
    }
  }

  return closestColor;
}
