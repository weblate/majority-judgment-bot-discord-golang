package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/sarulabs/di/v2"
	"log"
	"log/slog"
	"main/src/container"
	"os"
	"os/exec"
	"time"
)

// Rasterizer is the service that helps convert vector images to raster images.
type Rasterizer struct {
	logger    *slog.Logger
	resvgPath string
	tmpDir    string
}

func (service *Rasterizer) guessResvgPath() {
	service.SetResvgPath("./lib/resvg") // later on we could test multiple paths, maybe?
}

func (service *Rasterizer) guessTmpDir() {
	service.SetTmpDir("/tmp")
}

func (service *Rasterizer) SetTmpDir(path string) {
	service.tmpDir = path
}

func (service *Rasterizer) SetResvgPath(path string) {
	service.resvgPath = path
}

func (service *Rasterizer) ConvertSVGToPNG(
	ctx context.Context,
	svg []byte,
) (png []byte, err error) {

	nonce := time.Now().UnixNano()

	svgPath := fmt.Sprintf("%s/merit%d.svg", service.tmpDir, nonce)
	pngPath := fmt.Sprintf("%s/merit%d.png", service.tmpDir, nonce)

	err = os.WriteFile(svgPath, svg, 0664)
	if err != nil {
		return
	}

	_, err = os.Stat(svgPath)
	if err != nil {
		return
	}

	args := []string{
		svgPath,
		pngPath,
	}

	err = exec.CommandContext(ctx, service.resvgPath, args...).Run()
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			err = errors.New("timeout when running resvg")
		}
		return
	}

	_, err = os.Stat(pngPath)
	if err != nil {
		return
	}

	png, err = os.ReadFile(pngPath)
	if err != nil {
		return
	}

	err = os.Remove(svgPath)
	if err != nil {
		return
	}

	err = os.Remove(pngPath)
	if err != nil {
		return
	}

	return
}

// testSvg is used to ensure that this rasterizer works during init,
// in order to catch problems early, because it needs the resvg binary.
// We could also use go embedding for this, but I'm okay with this.
var testSvg = `<?xml version="1.0"?>
<svg width="800.00" height="208.00"
     viewBox="-16.00 -16.00 800.00 208.00"
     xmlns="http://www.w3.org/2000/svg">
<defs>
<clipPath id="clipProposal0" ><rect x="0.00" y="0.00" width="768.00" height="48.00" /></clipPath>
<clipPath id="clipProposal1" ><rect x="0.00" y="64.00" width="768.00" height="48.00" /></clipPath>
<clipPath id="clipProposal2" ><rect x="0.00" y="128.00" width="768.00" height="48.00" /></clipPath>
<pattern id="merit_pattern_nothing" x="0.00" y="0.00" width="1000.00" height="1000.00" patternUnits="userSpaceOnUse" >
</pattern>
<pattern id="merit_pattern_hexagons_1" x="0.00" y="0.00" width="13.86" height="16.00" patternUnits="userSpaceOnUse" >
<path d="M6.93 6.00 L5.20 7.00 L5.20 9.00 L6.93 10.00 L8.66 9.00 L8.66 7.00 ZM13.86 -2.00 L12.12 -1.00 L12.12 1.00 L13.86 2.00 L15.59 1.00 L15.59 -1.00 ZM0.00 -2.00 L-1.73 -1.00 L-1.73 1.00 L0.00 2.00 L1.73 1.00 L1.73 -1.00 ZM13.86 14.00 L12.12 15.00 L12.12 17.00 L13.86 18.00 L15.59 17.00 L15.59 15.00 ZM0.00 14.00 L-1.73 15.00 L-1.73 17.00 L0.00 18.00 L1.73 17.00 L1.73 15.00 Z" stroke="#000000" stroke-opacity="0.380" stroke-width="0.8" fill="none" />
</pattern>
<pattern id="merit_pattern_hexagons_2" x="0.00" y="0.00" width="13.86" height="16.00" patternUnits="userSpaceOnUse" >
<path d="M6.93 4.00 L3.46 6.00 L3.46 10.00 L6.93 12.00 L10.39 10.00 L10.39 6.00 ZM13.86 -4.00 L10.39 -2.00 L10.39 2.00 L13.86 4.00 L17.32 2.00 L17.32 -2.00 ZM0.00 -4.00 L-3.46 -2.00 L-3.46 2.00 L0.00 4.00 L3.46 2.00 L3.46 -2.00 ZM13.86 12.00 L10.39 14.00 L10.39 18.00 L13.86 20.00 L17.32 18.00 L17.32 14.00 ZM0.00 12.00 L-3.46 14.00 L-3.46 18.00 L0.00 20.00 L3.46 18.00 L3.46 14.00 Z" stroke="#000000" stroke-opacity="0.380" stroke-width="0.8" fill="none" />
</pattern>
<filter id="merit_filter_clean_outline" >
<feMorphology in="SourceAlpha" result="DILATED"  operator="dilate" radius="2 2" />
<feFlood result="COLOR"  flood-color="#ffffff" flood-opacity="1" />
<feComposite in="COLOR" in2="DILATED" result="OUTLINE"  operator="in" k1="0" k2="0" k3="0" k4="0" />
<feDropShadow in="OUTLINE" result="SHADOWED_OUTLINE" dx="0.618000" dy="1.000000" stdDeviation="0.5" flood-color="#000000" flood-opacity="1.000" />
<feMerge>
<feMergeNode in="SHADOWED_OUTLINE"/>
<feMergeNode in="SourceGraphic"/>
</feMerge>
</filter>
<filter id="merit_filter_blurry_outline" filterUnits="userSpaceOnUse" >
<feDropShadow dx="-1" dy="-1" stdDeviation="0.5" flood-color="#ffffff" flood-opacity="0.784" />
<feDropShadow dx="-1" dy="1" stdDeviation="0.5" flood-color="#ffffff" flood-opacity="0.784" />
<feDropShadow dx="1" dy="-1" stdDeviation="0.5" flood-color="#ffffff" flood-opacity="0.784" />
<feDropShadow dx="1" dy="1" stdDeviation="0.5" flood-color="#ffffff" flood-opacity="0.784" />
</filter>
</defs>
<rect x="-16.00" y="-16.00" width="800.00" height="208.00" rx="12.00" ry="12.00" fill="#000000" fill-opacity="1.000" />
<rect x="0.00" y="0.00" width="190.00" height="48.00" rx="6.00" ry="6.00" fill="#df3222" fill-opacity="1.000" />
<rect x="0.00" y="0.00" width="190.00" height="48.00" rx="6.00" ry="6.00" fill="url(#merit_pattern_nothing)" />
<rect x="194.00" y="0.00" width="149.60" height="48.00" rx="6.00" ry="6.00" fill="#fab001" fill-opacity="1.000" />
<rect x="194.00" y="0.00" width="149.60" height="48.00" rx="6.00" ry="6.00" fill="url(#merit_pattern_hexagons_1)" />
<rect x="347.60" y="0.00" width="420.40" height="48.00" rx="6.00" ry="6.00" fill="#00a249" fill-opacity="1.000" />
<rect x="347.60" y="0.00" width="420.40" height="48.00" rx="6.00" ry="6.00" fill="url(#merit_pattern_hexagons_2)" />
<line x1="384.00" y1="0.00" x2="384.00" y2="48.00" stroke="#010101" stroke-width="4" stroke-dasharray="6.472 4" filter="url(#merit_filter_blurry_outline)" />
<text x="16.00" y="24.00" style="clip-path: url(#clipProposal0)" stroke="#000000" stroke-opacity="1.000" fill="#000000" fill-opacity="1.000" font-size="1.618em" font-family="Noto Sans, Arial, Helvetica, sans-serif" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >Pizza 4 Dimensions<title>Pizza 4 Dimensions</title></text>
<text x="95.00" y="48.00" fill="#000000" fill-opacity="1.000" stroke="none" font-size="0.88em" font-family="Noto Sans, Arial, Helvetica, sans-serif" text-anchor="middle" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >5</text>
<text x="268.80" y="48.00" fill="#000000" fill-opacity="1.000" stroke="none" font-size="0.88em" font-family="Noto Sans, Arial, Helvetica, sans-serif" text-anchor="middle" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >4</text>
<text x="557.80" y="48.00" fill="#000000" fill-opacity="1.000" stroke="none" font-size="0.88em" font-family="Noto Sans, Arial, Helvetica, sans-serif" text-anchor="middle" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >11</text>
<rect x="0.00" y="64.00" width="343.60" height="48.00" rx="6.00" ry="6.00" fill="#df3222" fill-opacity="1.000" />
<rect x="0.00" y="64.00" width="343.60" height="48.00" rx="6.00" ry="6.00" fill="url(#merit_pattern_nothing)" />
<rect x="347.60" y="64.00" width="188.00" height="48.00" rx="6.00" ry="6.00" fill="#fab001" fill-opacity="1.000" />
<rect x="347.60" y="64.00" width="188.00" height="48.00" rx="6.00" ry="6.00" fill="url(#merit_pattern_hexagons_1)" />
<rect x="539.60" y="64.00" width="228.40" height="48.00" rx="6.00" ry="6.00" fill="#00a249" fill-opacity="1.000" />
<rect x="539.60" y="64.00" width="228.40" height="48.00" rx="6.00" ry="6.00" fill="url(#merit_pattern_hexagons_2)" />
<line x1="384.00" y1="64.00" x2="384.00" y2="112.00" stroke="#010101" stroke-width="4" stroke-dasharray="6.472 4" filter="url(#merit_filter_blurry_outline)" />
<text x="16.00" y="88.00" style="clip-path: url(#clipProposal1)" stroke="#000000" stroke-opacity="1.000" fill="#000000" fill-opacity="1.000" font-size="1.618em" font-family="Noto Sans, Arial, Helvetica, sans-serif" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >Lasagnes Assange<title>Lasagnes Assange</title></text>
<text x="171.80" y="112.00" fill="#000000" fill-opacity="1.000" stroke="none" font-size="0.88em" font-family="Noto Sans, Arial, Helvetica, sans-serif" text-anchor="middle" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >9</text>
<text x="441.60" y="112.00" fill="#000000" fill-opacity="1.000" stroke="none" font-size="0.88em" font-family="Noto Sans, Arial, Helvetica, sans-serif" text-anchor="middle" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >5</text>
<text x="653.80" y="112.00" fill="#000000" fill-opacity="1.000" stroke="none" font-size="0.88em" font-family="Noto Sans, Arial, Helvetica, sans-serif" text-anchor="middle" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >6</text>
<rect x="0.00" y="128.00" width="535.60" height="48.00" rx="6.00" ry="6.00" fill="#df3222" fill-opacity="1.000" />
<rect x="0.00" y="128.00" width="535.60" height="48.00" rx="6.00" ry="6.00" fill="url(#merit_pattern_nothing)" />
<rect x="539.60" y="128.00" width="228.40" height="48.00" rx="6.00" ry="6.00" fill="#00a249" fill-opacity="1.000" />
<rect x="539.60" y="128.00" width="228.40" height="48.00" rx="6.00" ry="6.00" fill="url(#merit_pattern_hexagons_2)" />
<line x1="384.00" y1="128.00" x2="384.00" y2="176.00" stroke="#010101" stroke-width="4" stroke-dasharray="6.472 4" filter="url(#merit_filter_blurry_outline)" />
<text x="16.00" y="152.00" style="clip-path: url(#clipProposal2)" stroke="#000000" stroke-opacity="1.000" fill="#000000" fill-opacity="1.000" font-size="1.618em" font-family="Noto Sans, Arial, Helvetica, sans-serif" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >Jurassique Pâtes<title>Jurassique Pâtes</title></text>
<text x="267.80" y="176.00" fill="#000000" fill-opacity="1.000" stroke="none" font-size="0.88em" font-family="Noto Sans, Arial, Helvetica, sans-serif" text-anchor="middle" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >14</text>
<text x="653.80" y="176.00" fill="#000000" fill-opacity="1.000" stroke="none" font-size="0.88em" font-family="Noto Sans, Arial, Helvetica, sans-serif" text-anchor="middle" dominant-baseline="middle" filter="url(#merit_filter_clean_outline)" >6</text>
</svg>`

func (service *Rasterizer) Test() error {
	ctx, cancel := context.WithTimeout(context.TODO(), 120*time.Second)
	defer cancel()
	_, err := service.ConvertSVGToPNG(ctx, []byte(testSvg))
	return err
}

func init() {
	err := container.GetBuilder().Add(di.Def{
		Name: "rasterizer",
		Build: func(ctn di.Container) (interface{}, error) {
			service := &Rasterizer{
				logger: ctn.Get("logger").(*slog.Logger),
			}
			service.guessResvgPath()
			service.guessTmpDir()
			return service, nil
		},
	})
	if err != nil {
		log.Fatalln("rasterizer failed to build", err)
	}
}
