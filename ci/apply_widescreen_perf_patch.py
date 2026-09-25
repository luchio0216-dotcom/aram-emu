#!/usr/bin/env python3
"""Apply the experimental ARAM widescreen rendering performance patch.

This keeps the product repository self-contained while the test build still pins
an upstream aram-core commit. The patch is deliberately narrow:

1. RGB565 color-key sprites get an unscaled span-copy fast path.
2. Non-self slow blits stop allocating a full Color scratch buffer.
3. Regression tests are written into aram-core and run before the Android build.
"""

from __future__ import annotations

import pathlib
import sys


HELPER = r'''
// blitTransparentRGB565FastPath copies an unscaled RGB565 color-key sprite in
// opaque horizontal runs. KTF/WIPI sprites use this shape heavily: the old
// path decoded, compared, blended and re-encoded every pixel even though an
// opaque RGB565 source under RasterCopy is byte-for-byte identical at the
// destination. Transparent runs are skipped and opaque runs use copy().
//
// The descriptor's key has to be exactly representable in RGB565. The scalar
// path compares the decoded source color with the descriptor Color, so using a
// rounded raw key for a non-canonical Color would change semantics.
func blitTransparentRGB565FastPath(
	destination, source *surface,
	destinationRectangle, sourceRectangle Rectangle,
) bool {
	if destination == source ||
		destinationRectangle.Width != sourceRectangle.Width ||
		destinationRectangle.Height != sourceRectangle.Height ||
		destination.descriptor.Format != PixelRGB565 ||
		source.descriptor.Format != PixelRGB565 ||
		source.descriptor.Transparent == nil {
		return false
	}
	if destination.state.Transparency && destination.descriptor.Transparent != nil {
		return false
	}
	if destination.state.Raster != RasterCopy ||
		destination.state.GlobalAlpha != 0xff ||
		destination.state.GlobalTransparency256 != 0 {
		return false
	}

	key := *source.descriptor.Transparent
	if key.A != 0xff ||
		expand5(key.R>>3) != key.R ||
		expand6(key.G>>2) != key.G ||
		expand5(key.B>>3) != key.B {
		return false
	}
	keyValue := uint16(key.R>>3)<<11 |
		uint16(key.G>>2)<<5 |
		uint16(key.B>>3)
	keyLow, keyHigh := byte(keyValue), byte(keyValue>>8)

	originLeft := int64(destinationRectangle.X) + int64(destination.state.TranslateX)
	originTop := int64(destinationRectangle.Y) + int64(destination.state.TranslateY)
	clip := destination.state.Clip
	left := max(originLeft, max(0, int64(clip.X)))
	top := max(originTop, max(0, int64(clip.Y)))
	right := min(
		originLeft+int64(destinationRectangle.Width),
		min(int64(destination.descriptor.Width), clip.Right()),
	)
	bottom := min(
		originTop+int64(destinationRectangle.Height),
		min(int64(destination.descriptor.Height), clip.Bottom()),
	)
	if right <= left || bottom <= top {
		return true
	}

	sourceStride := int(source.descriptor.Stride)
	destinationStride := int(destination.descriptor.Stride)
	sourceLeft := int64(sourceRectangle.X) + left - originLeft
	sourceTop := int64(sourceRectangle.Y) + top - originTop
	width := int(right - left)

	var dirtyLeft, dirtyTop, dirtyRight, dirtyBottom int64
	haveDirty := false
	for row := int64(0); row < bottom-top; row++ {
		sourceOffset := int(sourceTop+row)*sourceStride + int(sourceLeft)*2
		destinationOffset := int(top+row)*destinationStride + int(left)*2
		for x := 0; x < width; {
			offset := sourceOffset + x*2
			if source.pixels[offset] == keyLow && source.pixels[offset+1] == keyHigh {
				x++
				continue
			}
			runStart := x
			x++
			for x < width {
				offset = sourceOffset + x*2
				if source.pixels[offset] == keyLow && source.pixels[offset+1] == keyHigh {
					break
				}
				x++
			}
			runBytes := (x - runStart) * 2
			copy(
				destination.pixels[destinationOffset+runStart*2:destinationOffset+runStart*2+runBytes],
				source.pixels[sourceOffset+runStart*2:sourceOffset+runStart*2+runBytes],
			)

			runLeft := left + int64(runStart)
			runRight := left + int64(x)
			runTop := top + row
			if !haveDirty {
				dirtyLeft, dirtyRight = runLeft, runRight
				dirtyTop, dirtyBottom = runTop, runTop+1
				haveDirty = true
			} else {
				dirtyLeft = min(dirtyLeft, runLeft)
				dirtyRight = max(dirtyRight, runRight)
				dirtyTop = min(dirtyTop, runTop)
				dirtyBottom = max(dirtyBottom, runTop+1)
			}
		}
	}
	if haveDirty {
		destination.dirty = destination.dirty.Union(Rectangle{
			X:      int32(dirtyLeft),
			Y:      int32(dirtyTop),
			Width:  int32(dirtyRight - dirtyLeft),
			Height: int32(dirtyBottom - dirtyTop),
		})
	}
	return true
}

'''


TESTS = r'''package runtime

import "testing"

func TestTransparentRGB565FastPathMatchesReference(t *testing.T) {
	key := RGB(0, 0, 0)
	destination := bulkSurface(t, 28, 22, 6, PixelRGB565)
	source := bulkSurface(t, 18, 14, 4, PixelRGB565)
	source.descriptor.Transparent = &key
	for y := int32(0); y < source.descriptor.Height; y++ {
		for x := int32(0); x < source.descriptor.Width; x++ {
			if (x+y)%5 == 0 || (x >= 7 && x <= 9 && y%3 == 0) {
				encodeSurfaceColor(source, x, y, key)
				continue
			}
			encodeSurfaceColor(source, x, y, RGB(
				uint8(40+x*8),
				uint8(60+y*10),
				uint8(80+(x+y)*4),
			))
		}
	}
	destination.state.TranslateX = 2
	destination.state.TranslateY = 1
	destination.state.Clip = Rectangle{X: 4, Y: 3, Width: 18, Height: 14}
	destinationRectangle := Rectangle{X: 1, Y: 2, Width: 14, Height: 10}
	sourceRectangle := Rectangle{X: 2, Y: 1, Width: 14, Height: 10}
	slow := cloneSurface(destination)
	slowSource := cloneSurface(source)
	if !blitTransparentRGB565FastPath(destination, source, destinationRectangle, sourceRectangle) {
		t.Fatal("RGB565 color-key blit did not take the span fast path")
	}
	check(t, referenceScaledBlit(slow, slowSource, destinationRectangle, sourceRectangle))
	compareSurfaces(t, "transparent RGB565 span copy", destination, slow)
}

func TestTransparentRGB565FastPathRejectsNonCanonicalKey(t *testing.T) {
	key := Color{R: 1, G: 2, B: 3, A: 0xff}
	destination := bulkSurface(t, 8, 8, 0, PixelRGB565)
	source := bulkSurface(t, 8, 8, 0, PixelRGB565)
	source.descriptor.Transparent = &key
	if blitTransparentRGB565FastPath(destination, source, Rectangle{Width: 8, Height: 8}, Rectangle{Width: 8, Height: 8}) {
		t.Fatal("non-canonical RGB565 key must stay on the scalar path")
	}
}

func TestScaledBlitTransparentRGB565UsesBulkPath(t *testing.T) {
	graphics, err := NewGraphics(NewRegistry(8), GraphicsLimits{})
	check(t, err)
	key := RGB(0, 0, 0)
	destinationID, err := graphics.CreateSurface(1, SurfaceDescriptor{Width: 32, Height: 24, Format: PixelRGB565})
	check(t, err)
	sourceID, err := graphics.CreateSurface(1, SurfaceDescriptor{Width: 16, Height: 16, Format: PixelRGB565, Transparent: &key})
	check(t, err)
	source, err := graphics.get(sourceID, 1)
	check(t, err)
	fillSurface(source, key)
	for y := int32(3); y < 13; y++ {
		for x := int32(4); x < 12; x++ {
			encodeSurfaceColor(source, x, y, RGB(90, 150, 210))
		}
	}
	before := graphics.bulkFastPathCalls
	check(t, graphics.Blit(1, destinationID, sourceID, 5, 4, Rectangle{Width: 16, Height: 16}))
	if graphics.bulkFastPathCalls != before+1 {
		t.Fatalf("RGB565 transparent blit fast-path count = %d, want %d", graphics.bulkFastPathCalls, before+1)
	}
}

func TestScaledBlitStreamingSlowPathMatchesReference(t *testing.T) {
	graphics, err := NewGraphics(NewRegistry(8), GraphicsLimits{})
	check(t, err)
	destinationID, err := graphics.CreateSurface(1, SurfaceDescriptor{Width: 20, Height: 18, Format: PixelRGBA8888})
	check(t, err)
	sourceID, err := graphics.CreateSurface(1, SurfaceDescriptor{Width: 12, Height: 10, Format: PixelRGBA8888})
	check(t, err)
	destination, err := graphics.get(destinationID, 1)
	check(t, err)
	source, err := graphics.get(sourceID, 1)
	check(t, err)
	for index := range destination.pixels { destination.pixels[index] = byte(index*19 + 3) }
	for index := range source.pixels { source.pixels[index] = byte(index*31 + 11) }
	makeOpaque(source)
	destination.state.GlobalAlpha = 0x80
	destinationRectangle := Rectangle{X: 2, Y: 3, Width: 14, Height: 12}
	sourceRectangle := Rectangle{X: 1, Y: 1, Width: 9, Height: 7}
	slow := cloneSurface(destination)
	slowSource := cloneSurface(source)
	check(t, graphics.ScaledBlit(1, destinationID, sourceID, destinationRectangle, sourceRectangle))
	check(t, referenceScaledBlit(slow, slowSource, destinationRectangle, sourceRectangle))
	compareSurfaces(t, "streaming non-self slow path", destination, slow)
}
'''


def replace_once(text: str, old: str, new: str, label: str) -> str:
    if old not in text:
        raise SystemExit(f"{label}: expected source block was not found")
    if text.count(old) != 1:
        raise SystemExit(f"{label}: expected source block appears {text.count(old)} times")
    return text.replace(old, new, 1)


def main() -> None:
    if len(sys.argv) != 2:
        raise SystemExit("usage: apply_widescreen_perf_patch.py <aram-core-dir>")
    root = pathlib.Path(sys.argv[1]).resolve()
    graphics = root / "runtime" / "graphics.go"
    bulk = root / "runtime" / "graphics_bulk.go"
    tests = root / "runtime" / "widescreen_fastpath_test.go"

    graphics_text = graphics.read_text()
    old = '''\tif blitFastPath(destination, source, destinationRectangle, sourceRectangle) {\n\t\tg.bulkFastPathCalls++\n\t\treturn nil\n\t}\n\tcolors := make([]Color, int(count))\n'''
    new = '''\tif blitFastPath(destination, source, destinationRectangle, sourceRectangle) {\n\t\tg.bulkFastPathCalls++\n\t\treturn nil\n\t}\n\tif blitTransparentRGB565FastPath(destination, source, destinationRectangle, sourceRectangle) {\n\t\tg.bulkFastPathCalls++\n\t\treturn nil\n\t}\n\tif destination != source {\n\t\tfor y := int32(0); y < destinationRectangle.Height; y++ {\n\t\t\tsourceY := sourceRectangle.Y +\n\t\t\t\tint32(int64(y)*int64(sourceRectangle.Height)/int64(destinationRectangle.Height))\n\t\t\tfor x := int32(0); x < destinationRectangle.Width; x++ {\n\t\t\t\tsourceX := sourceRectangle.X +\n\t\t\t\t\tint32(int64(x)*int64(sourceRectangle.Width)/int64(destinationRectangle.Width))\n\t\t\t\tcolor := decodeSurfaceColor(source, sourceX, sourceY)\n\t\t\t\tif source.descriptor.Transparent != nil && color == *source.descriptor.Transparent {\n\t\t\t\t\tcontinue\n\t\t\t\t}\n\t\t\t\tif err := drawSurfacePixel(destination, destinationRectangle.X+x, destinationRectangle.Y+y, color); err != nil {\n\t\t\t\t\treturn err\n\t\t\t\t}\n\t\t\t}\n\t\t}\n\t\treturn nil\n\t}\n\tcolors := make([]Color, int(count))\n'''
    graphics.write_text(replace_once(graphics_text, old, new, "graphics.go"))

    bulk_text = bulk.read_text()
    marker = "// maskPixelByte applies mask to one byte of every pixel in the region.\n"
    if marker not in bulk_text:
        raise SystemExit("graphics_bulk.go: insertion marker not found")
    if "func blitTransparentRGB565FastPath(" not in bulk_text:
        bulk_text = bulk_text.replace(marker, HELPER + marker, 1)
    bulk.write_text(bulk_text)
    tests.write_text(TESTS)
    print("Applied widescreen performance patch to", root)


if __name__ == "__main__":
    main()
