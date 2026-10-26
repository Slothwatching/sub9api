// @vitest-environment node
import { describe, expect, it } from 'vitest'
import { execFileSync, spawnSync } from 'node:child_process'
import { mkdtempSync, readFileSync, rmSync, writeFileSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { dirname, join } from 'node:path'
import { fileURLToPath } from 'node:url'

const example = fileURLToPath(new URL('../../../public/guides/thesis-example/', import.meta.url))
const files = ['AGENTS.md', 'README.md', 'review-checklist.md', 'index.html', 'styles.css', 'app.js', 'data/review.json', 'data/review.template.json', 'scripts/validate-review.js', 'scripts/preview.js']

describe('downloaded no-dependency thesis example', () => {
  it('ships exactly the current files and runs standalone after ZIP extraction', () => {
    const dir = mkdtempSync(join(tmpdir(), 'thesis-example-'))
    try {
      execFileSync('unzip', ['-q', join(dirname(example), 'thesis-example.zip'), '-d', dir])
      for (const file of files) expect(readFileSync(join(dir, file))).toEqual(readFileSync(join(example, file)))
      const output = execFileSync(process.execPath, [join(dir, 'scripts/validate-review.js')], { cwd: tmpdir(), encoding: 'utf8' })
      expect(output).toContain('结构校验通过：5 篇')
      expect(output).toContain('人工核对 DOI 和原文')
      expect(JSON.parse(readFileSync(join(dir, 'data/review.template.json'), 'utf8')).papers).toEqual([])
    } finally { rmSync(dir, { recursive: true, force: true }) }
  })

  it('rejects plausible but incomplete or malformed data, without modifying the shipped records', () => {
    const original = JSON.parse(readFileSync(join(example, 'data/review.json'), 'utf8'))
    const cases: Array<[string, (data: typeof original) => void]> = [
      ['DOI 格式无效', d => { d.papers[0].doi = 'not-a-doi' }],
      ['DOI 重复', d => { d.papers[1].doi = d.papers[0].doi }],
      ['核实日期无效', d => { d.papers[0].verified_at = '2026-02-30' }],
      ['研究设计', d => { delete d.papers[0].study_design }],
      ['证据来源', d => { d.papers[0].evidence_sources[0].url = 'javascript:alert(1)' }],
      ['引用计数', d => { d.papers[0].citation_counts.supporting = -1 }],
      ['更正声明', d => { d.papers[0].editorial_statements.correction = {} }],
      ['获取分类', d => { d.papers[0].full_text_access.status = '可购买 GenAI 使用权' }],
      ['GenAI', d => { delete d.papers[0].full_text_access.genai_usage_rights }],
      ['尚未收录', d => { d.papers = [] }],
    ]
    const dir = mkdtempSync(join(tmpdir(), 'thesis-validator-'))
    try {
      execFileSync('unzip', ['-q', join(dirname(example), 'thesis-example.zip'), '-d', dir])
      for (const [expected, mutate] of cases) {
        const data = structuredClone(original)
        mutate(data)
        writeFileSync(join(dir, 'data/review.json'), JSON.stringify(data))
        const result = spawnSync(process.execPath, [join(dir, 'scripts/validate-review.js')], { encoding: 'utf8' })
        expect(result.status).toBe(1)
        expect(result.stderr).toContain(expected)
      }
    } finally { rmSync(dir, { recursive: true, force: true }) }
  })
})
