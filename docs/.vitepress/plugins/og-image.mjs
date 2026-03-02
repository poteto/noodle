/**
 * Vendored from @nolebase/vitepress-plugin-og-image v2.18.2
 * https://github.com/nolebase/integrations
 *
 * MIT License
 * Copyright (c) 2023-PRESENT All the contributors of Nolebase
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 *
 * Changes from upstream:
 * - Skip external links (http/https) in sidebar when collecting pages
 * - Use loadSystemFonts: true (no bundled 16MB font file)
 */

import { resolve, dirname, relative, join, sep, basename } from "node:path";
import { sep as posixSep } from "node:path/posix";
import fs from "fs-extra";
import GrayMatter from "gray-matter";
import RehypeMeta from "rehype-meta";
import RehypeParse from "rehype-parse";
import RehypeStringify from "rehype-stringify";
import { cyan, gray, yellow, red, green } from "colorette";
import { defu } from "defu";
import { glob } from "tinyglobby";
import { unified } from "unified";
import { visit } from "unist-util-visit";
import { fileURLToPath } from "node:url";
import { Buffer } from "node:buffer";
import { readFile } from "node:fs/promises";
import { createRequire } from "node:module";
import { initWasm, Resvg } from "@resvg/resvg-wasm";
import regexCreator from "emoji-regex";
import ora from "ora";

const logModulePrefix = `${cyan("@nolebase/vitepress-plugin-og-image")}${gray(":")}`;

async function tryToLocateTemplateSVGFile(siteConfig, configTemplateSvgPath) {
  if (configTemplateSvgPath != null)
    return resolve(siteConfig.srcDir, configTemplateSvgPath);
  const templateSvgPathUnderPublicDir = resolve(
    siteConfig.srcDir,
    "public",
    "og-template.svg"
  );
  if (await fs.pathExists(templateSvgPathUnderPublicDir))
    return templateSvgPathUnderPublicDir;
  const __dirname = dirname(fileURLToPath(import.meta.url));
  const templateSvgPathUnderRootDir = resolve(
    __dirname,
    "assets",
    "og-template.svg"
  );
  if (await fs.pathExists(templateSvgPathUnderRootDir))
    return templateSvgPathUnderRootDir;
}

async function tryToLocateFontFile(siteConfig) {
  const fontPath = resolve(siteConfig.srcDir, "public", "JetBrainsMono.ttf");
  if (await fs.pathExists(fontPath)) return fontPath;
}

let fontBuffer;
async function initFontBuffer(fontPath) {
  if (!fontPath) return;
  if (fontBuffer) return fontBuffer;
  try {
    fontBuffer = await readFile(fontPath);
  } catch (err) {
    throw new Error(`Failed to read font file due to ${err}`);
  }
  return fontBuffer;
}

async function applyCategoryText(pageItem, categoryOptions) {
  if (typeof categoryOptions?.byCustomGetter !== "undefined") {
    const gotTextMaybePromise = categoryOptions.byCustomGetter({
      ...pageItem,
    });
    if (typeof gotTextMaybePromise !== "undefined") {
      if (gotTextMaybePromise instanceof Promise)
        return await gotTextMaybePromise;
      if (gotTextMaybePromise) return gotTextMaybePromise;
    }
  }
  if (typeof categoryOptions?.byPathPrefix !== "undefined") {
    for (const { prefix, text } of categoryOptions.byPathPrefix) {
      if (pageItem.normalizedSourceFilePath.startsWith(prefix)) {
        if (!text) {
          console.warn(
            `${logModulePrefix} ${yellow("[WARN]")} empty text for prefix ${prefix} when processing ${pageItem.sourceFilePath} with categoryOptions.byPathPrefix, will ignore...`
          );
          return;
        }
        return text;
      }
      if (pageItem.normalizedSourceFilePath.startsWith(`/${prefix}`)) {
        if (!text) {
          console.warn(
            `${logModulePrefix} ${yellow("[WARN]")} empty text for prefix ${prefix} when processing ${pageItem.sourceFilePath} with categoryOptions.byPathPrefix, will ignore...`
          );
          return;
        }
        return text;
      }
    }
    console.warn(
      `${logModulePrefix} ${yellow("[WARN]")} no path prefix matched for ${pageItem.sourceFilePath} with categoryOptions.byPathPrefix, will ignore...`
    );
    return;
  }
  if (typeof categoryOptions?.byLevel !== "undefined") {
    const level = Number.parseInt(String(categoryOptions?.byLevel ?? 0));
    if (Number.isNaN(level)) {
      console.warn(
        `${logModulePrefix} ${yellow("[ERROR]")} byLevel must be a number, but got ${categoryOptions.byLevel} instead when processing ${pageItem.sourceFilePath} with categoryOptions.byLevel, will ignore...`
      );
      return;
    }
    const dirs = pageItem.sourceFilePath.split("/");
    if (dirs.length > level) return dirs[level];
    console.warn(
      `${logModulePrefix} ${red(`[ERROR] byLevel is out of range for ${pageItem.sourceFilePath} with categoryOptions.byLevel.`)} will ignore...`
    );
  }
}

async function applyCategoryTextWithFallback(pageItem, categoryOptions) {
  const customText = await applyCategoryText(pageItem, categoryOptions);
  if (customText) return customText;
  const fallbackWithFrontmatter =
    typeof categoryOptions?.fallbackWithFrontmatter === "undefined"
      ? true
      : categoryOptions.fallbackWithFrontmatter;
  if (
    fallbackWithFrontmatter &&
    "category" in pageItem.frontmatter &&
    pageItem.frontmatter.category &&
    typeof pageItem.frontmatter.category === "string"
  ) {
    return pageItem.frontmatter.category ?? "";
  }
  console.warn(
    `${logModulePrefix} ${yellow("[WARN]")} no category text found for ${pageItem.sourceFilePath} with categoryOptions ${JSON.stringify(categoryOptions)}.}`
  );
  return "Un-categorized";
}

const emojiRegex = regexCreator();
function removeEmoji(str) {
  return str.replace(emojiRegex, "");
}

const escapeMap = {
  "<": "&lt;",
  ">": "&gt;",
  "'": "&apos;",
  '"': "&quot;",
  "&": "&amp;",
};
function escape(content, ignore) {
  ignore = (ignore || "").replace(/[^&"<>']/g, "");
  const pattern = `([&"<>'])`.replace(new RegExp(`[${ignore}]`, "g"), "");
  return content.replace(new RegExp(pattern, "g"), (_, item) => {
    return escapeMap[item];
  });
}

const imageBuffers = new Map();
function templateSVG(
  siteName,
  siteDescription,
  title,
  category,
  ogTemplate,
  maxCharactersPerLine
) {
  maxCharactersPerLine ??= 17;
  const lines = removeEmoji(title)
    .trim()
    .replaceAll("\r\n", "\n")
    .split("\n")
    .map((line) => line.trim());
  for (let i = 0; i < lines.length; i++) {
    const val = lines[i].trim();
    if (val.length > maxCharactersPerLine) {
      let breakPoint = val.lastIndexOf(" ", maxCharactersPerLine);
      if (breakPoint < 0) {
        for (
          let j = Math.min(val.length - 1, maxCharactersPerLine);
          j > 0;
          j--
        ) {
          if (val[j] === val[j].toUpperCase()) {
            breakPoint = j;
            break;
          }
        }
      }
      if (breakPoint < 0) breakPoint = maxCharactersPerLine;
      lines[i] = val.slice(0, breakPoint);
      lines[i + 1] = `${val.slice(lines[i].length)}${lines[i + 1] || ""}`;
    }
    lines[i] = lines[i].trim();
  }
  const categoryStr = category ? removeEmoji(category).trim() : "";
  const data = {
    siteName,
    siteDescription,
    category: categoryStr,
    line1: lines[0] || "",
    line2: lines[1] || "",
    line3: `${lines[2] || ""}${lines[3] ? "..." : ""}`,
  };
  return ogTemplate.replace(/\{\{([^}]+)\}\}/g, (_, name) => {
    if (!name || typeof name !== "string" || !(name in data)) return "";
    const nameKeyOf = name;
    return escape(data[nameKeyOf]);
  });
}

let resvgInit = false;
async function initSVGRenderer() {
  try {
    if (!resvgInit) {
      const wasm = readFile(
        createRequire(import.meta.url).resolve(
          "@resvg/resvg-wasm/index_bg.wasm"
        )
      );
      await initWasm(wasm);
      resvgInit = true;
    }
  } catch (err) {
    throw new Error(`Failed to init resvg wasm due to ${err}`);
  }
}

async function renderSVG(
  svgContent,
  fontBuffer,
  imageUrlResolver,
  additionalFontBuffers,
  resultImageWidth
) {
  try {
    const resvg = new Resvg(svgContent, {
      fitTo: { mode: "width", value: resultImageWidth ?? 1200 },
      font: {
        fontBuffers: fontBuffer
          ? [fontBuffer, ...(additionalFontBuffers ?? [])]
          : additionalFontBuffers ?? [],
        loadSystemFonts: false,
      },
    });
    try {
      const resolvedImages = await Promise.all(
        resvg.imagesToResolve().map(async (url) => {
          return {
            url,
            buffer: await resolveImageUrlWithCache(url, imageUrlResolver),
          };
        })
      );
      for (const { url, buffer } of resolvedImages)
        resvg.resolveImage(url, buffer);
      const res = resvg.render();
      return {
        png: res.asPng(),
        width: res.width,
        height: res.height,
      };
    } catch (err) {
      throw new Error(
        `Failed to render open graph images on path due to ${err}`
      );
    }
  } catch (err) {
    throw new Error(
      `Failed to initiate Resvg instance to render open graph images due to ${err}`
    );
  }
}

function resolveImageUrlWithCache(url, imageUrlResolver) {
  if (imageBuffers.has(url)) return imageBuffers.get(url);
  const result = resolveImageUrl(url, imageUrlResolver);
  imageBuffers.set(url, result);
  return result;
}

async function resolveImageUrl(url, imageUrlResolver) {
  if (imageUrlResolver != null) {
    const res2 = await imageUrlResolver(url);
    if (res2 != null) return res2;
  }
  const res = await fetch(url);
  const buffer = await res.arrayBuffer();
  return Buffer.from(buffer);
}

const okMark = green("\u2713");
const failMark = red("\u2716");
async function task(taskName, task2) {
  const startsAt = Date.now();
  const moduleNamePrefix = cyan("@nolebase/vitepress-plugin-og-image");
  const grayPrefix = gray(":");
  const spinnerPrefix = `${moduleNamePrefix}${grayPrefix}`;
  const spinner = ora({ discardStdin: false });
  spinner.start(`${spinnerPrefix} ${taskName}...`);
  let result;
  try {
    result = await task2();
  } catch (e) {
    spinner.stopAndPersist({ symbol: failMark });
    throw e;
  }
  const elapsed = Date.now() - startsAt;
  const suffixText = `${gray(`(${elapsed}ms)`)} ${result || ""}`;
  spinner.stopAndPersist({ symbol: okMark, suffixText });
}

function renderTaskResultsSummary(results, siteConfig) {
  const successCount = results.filter((item) => item.status === "success");
  const skippedCount = results.filter((item) => item.status === "skipped");
  const erroredCount = results.filter((item) => item.status === "errored");
  const stats = `${green(`${successCount.length} generated`)}, ${yellow(`${skippedCount.length} skipped`)}, ${red(`${erroredCount.length} errored`)}`;
  const skippedList = ` - ${yellow("Following files were skipped")}:

${skippedCount
  .map((item) => {
    return gray(
      `    - ${relative(siteConfig.root, item.filePath)}: ${item.reason}`
    );
  })
  .join("\n")}`;
  const erroredList = ` - ${red("Following files encountered errors")}

${erroredCount
  .map((item) => {
    return gray(
      `    - ${relative(siteConfig.root, item.filePath)}: ${item.reason}`
    );
  })
  .join("\n")}`;
  const overallResults = [stats];
  if (skippedCount.length > 0) overallResults.push(skippedList);
  if (erroredCount.length > 0) overallResults.push(erroredList);
  return overallResults.join("\n\n");
}

function getLocales(siteData) {
  const locales = [];
  locales.push(siteData.lang ?? "root");
  if (Object.keys(siteData.locales).length === 0) return locales;
  for (const locale in siteData.locales) {
    if (locale !== siteData.lang) locales.push(locale);
  }
  return locales;
}

function getTitleWithLocales(siteData, locale) {
  if (Object.keys(siteData.locales).length > 0) {
    const title = siteData.locales[locale]?.title;
    if (title) return title;
    if (siteData.locales.root.title) return siteData.locales.root.title;
    return siteData.title;
  }
  return siteData.title;
}

function getDescriptionWithLocales(siteData, locale) {
  if (Object.keys(siteData.locales).length > 0) {
    const description = siteData.locales[locale]?.description;
    if (description) return description;
    if (siteData.locales.root.description)
      return siteData.locales.root.description;
    return siteData.description;
  }
  return siteData.description;
}

function getSidebar(siteData, themeConfig) {
  const locales = getLocales(siteData);
  if (locales.length === 0) {
    return {
      defaultLocale: siteData.lang,
      locales: locales || [],
      sidebar: {
        [siteData.lang]:
          flattenThemeConfigSidebar(themeConfig.sidebar) || [],
      },
    };
  }
  const sidebar = {
    defaultLocale: siteData.lang,
    locales,
    sidebar: {},
  };
  for (const locale of locales) {
    let themeConfigSidebar = [];
    if (
      typeof siteData.locales[locale]?.themeConfig?.sidebar !== "undefined"
    ) {
      if (Array.isArray(siteData.locales[locale]?.themeConfig?.sidebar)) {
        themeConfigSidebar =
          siteData.locales[locale]?.themeConfig?.sidebar || [];
      } else {
        themeConfigSidebar =
          siteData.locales[locale]?.themeConfig?.sidebar;
      }
    } else if (typeof siteData.themeConfig?.sidebar !== "undefined") {
      themeConfigSidebar = siteData.themeConfig?.sidebar || [];
    } else if (typeof themeConfig.sidebar !== "undefined") {
      themeConfigSidebar = themeConfig.sidebar;
    } else {
      themeConfigSidebar = [];
    }
    sidebar.sidebar[locale] =
      flattenThemeConfigSidebar(themeConfigSidebar) || [];
  }
  return sidebar;
}

function flattenThemeConfigSidebar(sidebar) {
  if (!sidebar) return [];
  if (Array.isArray(sidebar)) return sidebar;
  return Object.keys(sidebar).reduce((prev, curr) => {
    const items = sidebar[curr];
    return prev.concat(items);
  }, []);
}

function flattenSidebar(sidebar, base) {
  return sidebar.reduce((prev, curr) => {
    if (curr.items) {
      return prev.concat(
        flattenSidebar(
          curr.items.map((item) => addBaseToItem(item, curr.base ?? base)),
          curr.base ?? base
        ).concat(
          curr.link == null
            ? []
            : [
                {
                  ...curr,
                  items: void 0,
                  link:
                    curr.link != null
                      ? (curr.base ?? "") + curr.link
                      : curr.link,
                },
              ]
        )
      );
    }
    return prev.concat(curr);
  }, []);
}

function addBaseToItem(item, base) {
  if (base == null || base === "") return item;
  return {
    ...item,
    link: item.link != null ? base + item.link : item.link,
  };
}

function isExternalLink(link) {
  return link.startsWith("http://") || link.startsWith("https://");
}

async function renderSVGAndRewriteHTML(
  siteConfig,
  siteTitle,
  siteDescription,
  page,
  file,
  ogImageTemplateSvg,
  ogImageTemplateSvgPath,
  domain,
  imageUrlResolver,
  additionalFontBuffers,
  resultImageWidth,
  maxCharactersPerLine,
  overrideExistingMetaTags
) {
  const fileName = basename(file, ".html");
  const ogImageFilePathBaseName = `og-${fileName}.png`;
  const ogImageFilePathFullName = `${dirname(file)}/${ogImageFilePathBaseName}`;
  const html = await fs.readFile(file, "utf-8");
  const parsedHtml = unified()
    .use(RehypeParse, { fragment: true })
    .parse(html);
  let hasOgImage = false;
  visit(parsedHtml, "element", (node) => {
    if (
      node.tagName === "meta" &&
      (node.properties?.name === "og:image" ||
        node.properties?.name === "twitter:image")
    )
      hasOgImage = node.properties.name;
    else return true;
  });
  if (hasOgImage && !overrideExistingMetaTags) {
    return {
      filePath: file,
      status: "skipped",
      reason: `already has ${hasOgImage} meta tag`,
    };
  }
  const templatedOgImageSvg = templateSVG(
    siteTitle,
    siteDescription,
    page.title,
    page.category ?? "",
    ogImageTemplateSvg,
    maxCharactersPerLine
  );
  let width;
  let height;
  try {
    const res = await renderSVGAndSavePNG(
      templatedOgImageSvg,
      ogImageFilePathFullName,
      ogImageTemplateSvgPath,
      relative(siteConfig.srcDir, file),
      {
        fontPath: await tryToLocateFontFile(siteConfig),
        imageUrlResolver,
        additionalFontBuffers,
        resultImageWidth,
      }
    );
    width = res.width;
    height = res.height;
  } catch (err) {
    return {
      filePath: file,
      status: "errored",
      reason: String(err),
    };
  }
  const result = await unified()
    .use(RehypeParse)
    .use(RehypeMeta, {
      og: true,
      twitter: true,
      image: {
        url: `${domain}/${relative(siteConfig.outDir, ogImageFilePathFullName)
          .split(sep)
          .map((item) => encodeURIComponent(item))
          .join("/")}`,
        width,
        height,
      },
    })
    .use(RehypeStringify)
    .process(html);
  try {
    await fs.writeFile(file, String(result), "utf-8");
  } catch (err) {
    console.error(
      `${logModulePrefix} `,
      `${red("[ERROR] \u2717")} failed to write transformed HTML on path [${relative(siteConfig.srcDir, file)}] due to ${err}`,
      `\n${red(err.message)}\n${gray(String(err.stack))}`
    );
    return {
      filePath: file,
      status: "errored",
      reason: String(err),
    };
  }
  return {
    filePath: file,
    status: "success",
  };
}

async function renderSVGAndSavePNG(
  svgContent,
  saveAs,
  forSvgSource,
  forFile,
  options
) {
  try {
    const {
      png: pngBuffer,
      width,
      height,
    } = await renderSVG(
      svgContent,
      await initFontBuffer(options.fontPath),
      options.imageUrlResolver,
      options.additionalFontBuffers,
      options.resultImageWidth
    );
    try {
      await fs.writeFile(saveAs, pngBuffer, "binary");
    } catch (err) {
      console.error(
        `${logModulePrefix} `,
        `${red("[ERROR] \u2717")} open graph image rendered successfully, but failed to write generated open graph image on path [${saveAs}] due to ${err}`,
        `\n${red(err.message)}\n${gray(String(err.stack))}`
      );
      throw err;
    }
    return { width, height };
  } catch (err) {
    console.error(
      `${logModulePrefix} `,
      `${red("[ERROR] \u2717")} failed to generate open graph image as ${green(`[${saveAs}]`)} with ${green(`[${forSvgSource}]`)} due to ${red(String(err))}`,
      `skipped open graph image generation for ${green(`[${forFile}]`)}`,
      `\n\nSVG Content:\n\n${svgContent}`,
      `\n\nDetailed stack information bellow:\n\n${red(err.message)}\n${gray(String(err.stack))}`
    );
    throw err;
  }
}

export function buildEndGenerateOpenGraphImages(options) {
  options = defu(options, {
    resultImageWidth: 1200,
    maxCharactersPerLine: 17,
    overrideExistingMetaTags: true,
  });
  return async (siteConfig) => {
    await initSVGRenderer();
    const ogImageTemplateSvgPath = await tryToLocateTemplateSVGFile(
      siteConfig,
      options.templateSvgPath
    );
    await task("rendering open graph images", async () => {
      const themeConfig = siteConfig.site.themeConfig;
      const sidebar = getSidebar(siteConfig.site, themeConfig);
      let pages = [];
      for (const locale of sidebar.locales) {
        const flattenedSidebar = flattenSidebar(sidebar.sidebar[locale]);
        const items = [];
        for (const item of flattenedSidebar) {
          const relativeLink = item.link ?? "";

          // FIX: Skip external links and items without a link
          if (!relativeLink || isExternalLink(relativeLink)) {
            continue;
          }

          const sourceFilePath = relativeLink.endsWith("/")
            ? `${relativeLink}index.md`
            : relativeLink.endsWith(".md")
              ? relativeLink
              : `${relativeLink}.md`;
          const sourceFileContent = fs.readFileSync(
            `${join(siteConfig.srcDir, sourceFilePath)}`,
            "utf-8"
          );
          const { data } = GrayMatter(sourceFileContent);
          const res = {
            ...item,
            title: item.text ?? item.title ?? "Untitled",
            category: "",
            locale,
            frontmatter: data,
            sourceFilePath,
            normalizedSourceFilePath: sourceFilePath
              .split(sep)
              .join(posixSep),
          };
          res.category = await applyCategoryTextWithFallback(
            res,
            options.category
          );
          items.push(res);
        }
        pages = pages.concat(items);
      }
      const files = await glob(`${siteConfig.outDir}/**/*.html`, {
        onlyFiles: true,
      });
      if (!ogImageTemplateSvgPath) {
        return `${green(`${0} generated`)}, ${yellow(`${files.length} (all) skipped`)}, ${red(`${0} errored`)}.\n\n - ${red("Failed to locate")} og-template.svg ${red("under public or plugin directory")}, did you forget to put it? will skip open graph image generation.`;
      }
      const ogImageTemplateSvg = fs.readFileSync(
        ogImageTemplateSvgPath,
        "utf-8"
      );
      const generatedForFiles = await Promise.all(
        files.map(async (file) => {
          const relativePath = relative(siteConfig.outDir, file);
          const link = `/${relativePath.slice(0, relativePath.lastIndexOf(".")).replaceAll(sep, "/")}`
            .split("/index")[0];
          const page = pages.find((item) => {
            let itemLink = item.link;
            if (itemLink?.endsWith(".md"))
              itemLink = itemLink.slice(0, -".md".length);
            if (itemLink === link) return true;
            if (itemLink === `${link}/`) return true;
            return false;
          });
          if (!page) {
            return {
              filePath: file,
              status: "skipped",
              reason: "correspond Markdown page not found in sidebar",
            };
          }
          const siteTitle = getTitleWithLocales(siteConfig.site, page.locale);
          const siteDescription = getDescriptionWithLocales(
            siteConfig.site,
            page.locale
          );
          return await renderSVGAndRewriteHTML(
            siteConfig,
            siteTitle,
            siteDescription,
            page,
            file,
            ogImageTemplateSvg,
            ogImageTemplateSvgPath,
            options.baseUrl,
            options.svgImageUrlResolver,
            options.svgFontBuffers,
            options.resultImageWidth,
            options.maxCharactersPerLine,
            options.overrideExistingMetaTags
          );
        })
      );
      return renderTaskResultsSummary(generatedForFiles, siteConfig);
    });
  };
}
