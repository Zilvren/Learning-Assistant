# LaTeX 公式中的删除线

用 `\cancel` 命令，需要 `\require{cancel}` 或引入 cancel 包：

## 基础

```latex
\require{cancel}
\cancel{x^2} + \cancel{y^2} = \cancel{z^2}
```

效果：$\require{cancel}\cancel{x^2} + \cancel{y^2} = \cancel{z^2}$

## 变体

| 命令 | 效果 | 说明 |
|------|------|------|
| `\cancel{a+b}` | $\cancel{a+b}$ | 斜线删除 |
| `\bcancel{a+b}` | $\bcancel{a+b}$ | 反斜线删除 |
| `\xcancel{a+b}` | $\xcancel{a+b}$ | 叉号删除 |
| `\cancelto{0}{a+b}` | $\cancelto{0}{a+b}$ | 删除并替换 |

## 注意

你的项目用的是 KaTeX，KaTeX 默认支持 cancel，不用额外配置。直接在编辑器里写就行。
