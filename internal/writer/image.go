package writer

import (
	"github.com/fogleman/gg"
)

// TextImageConfig 文本图片配置
type TextImageConfig struct {
	Width          int     // 图片宽度
	FontPath       string  // 字体文件路径
	FontSize       float64 // 字体大小
	LineSpacing    float64 // 行间距
	TextWidthRatio float64 // 文本宽度占图片宽度的比例
	TopMargin      float64 // 顶部边距
	BottomMargin   float64 // 底部边距
	BackgroundPath string  // 背景图片路径
}

// GenerateTextImage 根据文本内容生成自适应高度的图片，通过平铺背景图片填充
func GenerateTextImage(text string, config TextImageConfig, outputPath string) error {
	// 创建临时context用于计算文本尺寸
	tempDC := gg.NewContext(config.Width, 100)
	if err := tempDC.LoadFontFace(config.FontPath, config.FontSize); err != nil {
		return err
	}

	// 计算需要的行数和每行文本
	maxTextWidth := float64(config.Width) * config.TextWidthRatio
	lineTexts := splitTextIntoLines(tempDC, text, maxTextWidth)

	// 计算字体高度和总文本高度
	fontHeight := tempDC.FontHeight()
	totalTextHeight := fontHeight*float64(len(lineTexts)) + config.LineSpacing*float64(len(lineTexts)-1)
	imageHeight := int(totalTextHeight + config.TopMargin + config.BottomMargin)

	// 创建实际的绘图context，使用计算出的高度
	dc := gg.NewContext(config.Width, imageHeight)
	dc.SetRGB(202.0/255, 235.0/255, 216.0/255) //背景天空蓝
	dc.Clear()

	// 加载并平铺背景图片
	if config.BackgroundPath != "" {
		backgroundImg, err := gg.LoadImage(config.BackgroundPath)
		if err != nil {
			return err
		}

		// 获取背景图片的高度
		bgHeight := backgroundImg.Bounds().Dy()

		// 垂直平铺背景图片
		for y := 0; y < imageHeight; y += bgHeight {
			dc.DrawImage(backgroundImg, 0, y)
		}
	}

	dc.SetRGB(0, 0, 0)
	if err := dc.LoadFontFace(config.FontPath, config.FontSize); err != nil {
		return err
	}

	// 从顶部边距开始绘制文本
	lineY := config.TopMargin + fontHeight
	for _, lineText := range lineTexts {
		textWidth, _ := dc.MeasureString(lineText)
		lineX := (float64(config.Width) - textWidth) / 2
		dc.DrawString(lineText, lineX, lineY)
		lineY += fontHeight + config.LineSpacing
	}

	return dc.SavePNG(outputPath)
}

// splitTextIntoLines 将文本分割成多行
func splitTextIntoLines(dc *gg.Context, text string, maxTextWidth float64) []string {
	lineTexts := make([]string, 0)
	remainingText := text

	for len(remainingText) > 0 {
		lineText := TruncateText(dc, remainingText, maxTextWidth)
		if lineText == "" {
			break
		}
		lineTexts = append(lineTexts, lineText)
		if len(lineText) >= len(remainingText) {
			break
		}
		remainingText = remainingText[len(lineText):]
	}

	return lineTexts
}

func TruncateText(dc *gg.Context, originalText string, maxTextWidth float64) string {
	tmpStr := ""
	result := make([]rune, 0)
	for _, r := range originalText {
		tmpStr = tmpStr + string(r)
		w, _ := dc.MeasureString(tmpStr)
		if w > maxTextWidth {
			if len(tmpStr) <= 1 {
				return ""
			} else {
				break
			}
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
