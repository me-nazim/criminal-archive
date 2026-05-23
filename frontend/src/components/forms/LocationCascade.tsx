// Cascading country → division → district → upazila selector.
//
// For Bangladesh we expose the full four-level hierarchy. For other
// countries only the country select is shown (BD has the only seeded
// sub-hierarchy). All four IDs are emitted via onChange so callers can
// store them as-is.

import { useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Select, type SelectOption } from '../ui/Select'
import {
  useCountries,
  useDistricts,
  useDivisions,
  useUpazilas,
  pickName,
} from '../../hooks/useReferenceData'
import { TextField } from '../ui/TextField'

export interface LocationValue {
  countryId: number | null
  divisionId: number | null
  districtId: number | null
  upazilaId: number | null
  /** Free-text location used for non-BD countries or further detail. */
  text: string
}

interface Props {
  value: LocationValue
  onChange: (next: LocationValue) => void
  errors?: Partial<Record<keyof LocationValue, string>>
  disabled?: boolean
}

const ISO2_BD = 'BD'

export function LocationCascade({ value, onChange, errors, disabled }: Props) {
  const { t, i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'bn'

  const countries = useCountries()
  const isBD = useMemo(() => {
    if (!countries.data || !value.countryId) return false
    return countries.data.find((c) => c.id === value.countryId)?.iso2 === ISO2_BD
  }, [countries.data, value.countryId])

  const divisions = useDivisions(isBD ? value.countryId : null)
  const districts = useDistricts(isBD ? value.divisionId : null)
  const upazilas = useUpazilas(isBD ? value.districtId : null)

  // When the country changes away from BD, drop the sub-IDs.
  useEffect(() => {
    if (!isBD && (value.divisionId || value.districtId || value.upazilaId)) {
      onChange({ ...value, divisionId: null, districtId: null, upazilaId: null })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isBD])

  const countryOptions: SelectOption[] = (countries.data ?? []).map((c) => ({
    value: c.id,
    label: pickName(c, locale) + (c.iso2 ? ` (${c.iso2})` : ''),
  }))

  const divisionOptions: SelectOption[] = (divisions.data ?? []).map((d) => ({
    value: d.id,
    label: pickName(d, locale),
  }))

  const districtOptions: SelectOption[] = (districts.data ?? []).map((d) => ({
    value: d.id,
    label: pickName(d, locale),
  }))

  const upazilaOptions: SelectOption[] = (upazilas.data ?? []).map((u) => ({
    value: u.id,
    label: pickName(u, locale),
  }))

  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <Select
        label={t('location.country')}
        placeholder={t('location.select_country') ?? ''}
        showRequired
        disabled={disabled || countries.isPending}
        options={countryOptions}
        value={value.countryId ?? ''}
        errorText={errors?.countryId}
        onChange={(e) => {
          const v = e.target.value === '' ? null : Number(e.target.value)
          onChange({
            ...value,
            countryId: v,
            divisionId: null,
            districtId: null,
            upazilaId: null,
          })
        }}
      />

      {isBD && (
        <>
          <Select
            label={t('location.division')}
            placeholder={t('location.select_division') ?? ''}
            disabled={disabled || !value.countryId || divisions.isPending}
            options={divisionOptions}
            value={value.divisionId ?? ''}
            errorText={errors?.divisionId}
            onChange={(e) => {
              const v = e.target.value === '' ? null : Number(e.target.value)
              onChange({ ...value, divisionId: v, districtId: null, upazilaId: null })
            }}
          />

          <Select
            label={t('location.district')}
            placeholder={t('location.select_district') ?? ''}
            disabled={disabled || !value.divisionId || districts.isPending}
            options={districtOptions}
            value={value.districtId ?? ''}
            errorText={errors?.districtId}
            onChange={(e) => {
              const v = e.target.value === '' ? null : Number(e.target.value)
              onChange({ ...value, districtId: v, upazilaId: null })
            }}
          />

          <Select
            label={t('location.upazila')}
            placeholder={t('location.select_upazila') ?? ''}
            disabled={disabled || !value.districtId || upazilas.isPending}
            options={upazilaOptions}
            value={value.upazilaId ?? ''}
            errorText={errors?.upazilaId}
            onChange={(e) => {
              const v = e.target.value === '' ? null : Number(e.target.value)
              onChange({ ...value, upazilaId: v })
            }}
          />
        </>
      )}

      <div className={isBD ? 'sm:col-span-2' : 'sm:col-span-2'}>
        <TextField
          label={t('location.text_label')}
          helperText={t('location.text_help') ?? ''}
          placeholder={t('location.text_placeholder') ?? ''}
          disabled={disabled}
          value={value.text}
          onChange={(e) => onChange({ ...value, text: e.target.value })}
        />
      </div>
    </div>
  )
}
