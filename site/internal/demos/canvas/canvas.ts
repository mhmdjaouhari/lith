// The canvas's controller, on the stage (data-canvas). sb-pixel-board
// paints with sixteen hex colours; the page reads them from the site's
// tokens (--canvas-c0 to -c15 in canvas.css, each with a light and a dark
// value), through a probe that resolves a token and a 1 by 1 canvas that
// turns any CSS colour into sRGB, and reads them again when the theme
// changes. Everything else the canvas does is markup (views.templ).

import { controller, must, signals } from 'lith-ui'
import { ClassProbe, Colours, PropColour, type Roots } from './ds.gen'

controller('canvas', (el, scope) => {
	const s = signals<Roots>()._canvas
	const probe = must(el, `.${ClassProbe}`)
	const pen = document.createElement('canvas').getContext('2d', { willReadFrequently: true })
	if (!pen) throw new Error('canvas: no 2d context')

	/** The palette in hex, as the board's palette attribute takes it. */
	const read = (): void => {
		s.palette = JSON.stringify(
			Array.from({ length: Colours }, (_, i) => {
				probe.style.color = `var(${PropColour}${i})`
				pen.fillStyle = getComputedStyle(probe).color
				pen.fillRect(0, 0, 1, 1)
				const [r = 0, g = 0, b = 0] = pen.getImageData(0, 0, 1, 1).data
				return '#' + ((1 << 24) | (r << 16) | (g << 8) | b).toString(16).slice(1)
			}),
		)
	}
	read()
	const theme = new MutationObserver(read)
	theme.observe(document.documentElement, { attributeFilter: ['data-lith-theme'] })
	scope.defer(() => theme.disconnect())
	const dark = matchMedia('(prefers-color-scheme: dark)')
	dark.addEventListener('change', read)
	scope.defer(() => dark.removeEventListener('change', read))
})
