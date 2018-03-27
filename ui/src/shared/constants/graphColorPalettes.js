import chroma from 'chroma-js'

const COLOR_TYPE_SCALE = 'scale'

// Color Palettes
export const LINE_COLORS_A = [
  {
    type: COLOR_TYPE_SCALE,
    hex: '#31C0F6',
    id: '0',
    name: 'Nineteen Eighty Four',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#A500A5',
    id: '0',
    name: 'Nineteen Eighty Four',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#FF7E27',
    id: '0',
    name: 'Nineteen Eighty Four',
    value: 0,
  },
]

export const LINE_COLORS_B = [
  {
    type: COLOR_TYPE_SCALE,
    hex: '#74D495',
    id: '1',
    name: 'Atlantis',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#3F3FBA',
    id: '1',
    name: 'Atlantis',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#EA5994',
    id: '1',
    name: 'Atlantis',
    value: 0,
  },
]

export const LINE_COLORS_C = [
  {
    type: COLOR_TYPE_SCALE,
    hex: '#8F8AF4',
    id: '1',
    name: 'Do Androids Dream of Electric Sheep?',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#A51414',
    id: '1',
    name: 'Do Androids Dream of Electric Sheep?',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#F4CF31',
    id: '1',
    name: 'Do Androids Dream of Electric Sheep?',
    value: 0,
  },
]

export const LINE_COLORS_D = [
  {
    type: COLOR_TYPE_SCALE,
    hex: '#FD7A5D',
    id: '1',
    name: 'Delorean',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#5F1CF2',
    id: '1',
    name: 'Delorean',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#4CE09A',
    id: '1',
    name: 'Delorean',
    value: 0,
  },
]

export const LINE_COLORS_E = [
  {
    type: COLOR_TYPE_SCALE,
    hex: '#FDC44F',
    id: '1',
    name: 'Cthulhu',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#007C76',
    id: '1',
    name: 'Cthulhu',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#8983FF',
    id: '1',
    name: 'Cthulhu',
    value: 0,
  },
]

export const LINE_COLORS_F = [
  {
    type: COLOR_TYPE_SCALE,
    hex: '#DA6FF1',
    id: '1',
    name: 'Ectoplasm',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#00717A',
    id: '1',
    name: 'Ectoplasm',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#ACFF76',
    id: '1',
    name: 'Ectoplasm',
    value: 0,
  },
]

export const LINE_COLORS_G = [
  {
    type: COLOR_TYPE_SCALE,
    hex: '#F6F6F8',
    id: '1',
    name: 'T-Max 400 Film',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#A4A8B6',
    id: '1',
    name: 'T-Max 400 Film',
    value: 0,
  },
  {
    type: COLOR_TYPE_SCALE,
    hex: '#545667',
    id: '1',
    name: 'T-Max 400 Film',
    value: 0,
  },
]

export const DEFAULT_LINE_COLORS = LINE_COLORS_A

export const LINE_COLOR_SCALES = [
  LINE_COLORS_A,
  LINE_COLORS_B,
  LINE_COLORS_C,
  LINE_COLORS_D,
  LINE_COLORS_E,
  LINE_COLORS_F,
  LINE_COLORS_G,
].map(colorScale => {
  const name = colorScale[0].name
  const colors = colorScale
  const id = colorScale[0].id

  return {name, colors, id}
})

export const validateLineColors = colors => {
  if (!colors || colors.length !== 3) {
    return DEFAULT_LINE_COLORS
  }

  const testColorsTypes =
    colors.filter(color => color.type === COLOR_TYPE_SCALE).length ===
    colors.length

  return testColorsTypes ? colors : DEFAULT_LINE_COLORS
}

export const getLineColorsHexes = (colors, numSeries) => {
  const validatedColors = validateLineColors(colors) // ensures safe defaults
  const colorsHexArray = validatedColors.map(color => color.hex)

  if (numSeries === 1) {
    return [colorsHexArray[0]]
  }
  if (numSeries === 2) {
    return [colorsHexArray[0], colorsHexArray[1]]
  }
  return chroma
    .scale(colorsHexArray)
    .mode('lch')
    .colors(numSeries)
}
