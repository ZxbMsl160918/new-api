/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { zodResolver } from '@hookform/resolvers/zod'
import { useEffect, useRef } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import * as z from 'zod'

import { JsonCodeEditor } from '@/components/json-code-editor'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'

import { SettingsForm } from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  formatJsonForTextarea,
  normalizeJsonString,
  validateJsonString,
} from './utils'

const schema = z.object({
  context_limit: z.object({
    model_context_limits: z.string().superRefine((value, ctx) => {
      const result = validateJsonString(value)
      if (!result.valid) {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: result.message || 'Invalid JSON',
        })
        return
      }
      try {
        const parsed: unknown = JSON.parse(normalizeJsonString(value))
        if (
          typeof parsed !== 'object' ||
          parsed === null ||
          Array.isArray(parsed)
        ) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            message: 'Must be a JSON object mapping model names to token limits',
          })
          return
        }
        for (const [model, limit] of Object.entries(
          parsed as Record<string, unknown>
        )) {
          const wildcardCount = (model.match(/\*/g) ?? []).length
          const invalidWildcard =
            model === '*' ||
            wildcardCount > 1 ||
            (wildcardCount === 1 && !model.endsWith('*'))
          if (
            !model ||
            invalidWildcard ||
            typeof limit !== 'number' ||
            !Number.isFinite(limit) ||
            limit <= 0 ||
            !Number.isInteger(limit)
          ) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              message: `Invalid limit for model "${model}": use a positive integer and an optional trailing * wildcard`,
            })
            return
          }
        }
      } catch {
        ctx.addIssue({
          code: z.ZodIssueCode.custom,
          message: 'Invalid JSON',
        })
      }
    }),
  }),
})

type ContextLimitFormValues = z.output<typeof schema>
type ContextLimitFormInput = z.input<typeof schema>

type FlatContextLimitSettings = {
  'context_limit.model_context_limits': string
}

type ContextLimitCardProps = {
  defaultValues: ContextLimitFormInput
}

export function ContextLimitCard({ defaultValues }: ContextLimitCardProps) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const normalizedDefaultsRef = useRef<FlatContextLimitSettings>({
    'context_limit.model_context_limits': normalizeJsonString(
      defaultValues.context_limit.model_context_limits
    ),
  })

  const buildFormDefaults = (values: ContextLimitFormInput) => ({
    context_limit: {
      model_context_limits: formatJsonForTextarea(
        values.context_limit.model_context_limits
      ),
    },
  })

  const form = useForm<ContextLimitFormInput, unknown, ContextLimitFormValues>({
    resolver: zodResolver(schema),
    defaultValues: buildFormDefaults(defaultValues),
  })

  useEffect(() => {
    normalizedDefaultsRef.current = {
      'context_limit.model_context_limits': normalizeJsonString(
        defaultValues.context_limit.model_context_limits
      ),
    }
    form.reset(buildFormDefaults(defaultValues))
  }, [defaultValues, form])

  const onSubmit = async (values: ContextLimitFormValues) => {
    const normalized: FlatContextLimitSettings = {
      'context_limit.model_context_limits': normalizeJsonString(
        values.context_limit.model_context_limits
      ),
    }

    const updates = (
      Object.keys(normalized) as Array<keyof FlatContextLimitSettings>
    ).filter((key) => normalized[key] !== normalizedDefaultsRef.current[key])

    if (updates.length === 0) {
      toast.info(t('No changes to save'))
      return
    }

    for (const key of updates) {
      await updateOption.mutateAsync({ key, value: normalized[key] })
    }
  }

  return (
    <SettingsSection title={t('Model Context Limits')}>
      <Form {...form}>
        {/* eslint-disable-next-line react-hooks/refs */}
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            isSaving={updateOption.isPending}
          />
          <FormField
            control={form.control}
            name='context_limit.model_context_limits'
            render={({ field }) => (
              <FormItem>
                <FormLabel>{t('Context Window Limits')}</FormLabel>
                <FormControl>
                  <JsonCodeEditor
                    value={field.value}
                    onChange={field.onChange}
                    name={field.name}
                    onBlur={field.onBlur}
                    textareaRef={field.ref}
                    aria-invalid={Boolean(
                      form.formState.errors.context_limit?.model_context_limits
                    )}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'Set per-model context window limits in tokens. Requests whose estimated input tokens plus max_tokens exceed the limit are rejected with 400. Models not listed are unlimited. A trailing * matches model name prefixes; exact matches take priority, then the longest prefix.'
                  )}{' '}
                  {t('Example')}{' '}
                  {`{ "deepseek-v4-pro": 300000, "gpt-5.6-luna*": 300000 }`}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
