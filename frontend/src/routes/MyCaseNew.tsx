// "Submit a new case" page. We keep this lean: the user fills basic
// info, hits Save Draft, and is redirected to the editor where they can
// link persons and upload evidence before submitting for review.

import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'

import { Container } from '../components/ui/Container'
import { Card, CardBody, CardHeader } from '../components/ui/Card'
import { Button } from '../components/ui/Button'
import { TextField } from '../components/ui/TextField'
import { TextArea } from '../components/ui/TextArea'
import { Select } from '../components/ui/Select'
import { LocationCascade, type LocationValue } from '../components/forms/LocationCascade'
import { useCrimeTypes } from '../hooks/useReferenceData'
import { useCreateCase } from '../hooks/useCases'
import { ApiError } from '../lib/api'

const emptyLocation: LocationValue = {
  countryId: null,
  divisionId: null,
  districtId: null,
  upazilaId: null,
  text: '',
}

export default function MyCaseNew() {
  const { t, i18n } = useTranslation()
  const isBN = (i18n.resolvedLanguage ?? 'bn').startsWith('bn')
  const nav = useNavigate()
  const create = useCreateCase()
  const crimes = useCrimeTypes()

  const [titleBN, setTitleBN] = useState('')
  const [titleEN, setTitleEN] = useState('')
  const [summaryBN, setSummaryBN] = useState('')
  const [summaryEN, setSummaryEN] = useState('')
  const [incidentDate, setIncidentDate] = useState('')
  const [crimeTypeID, setCrimeTypeID] = useState('')
  const [tags, setTags] = useState('')
  const [loc, setLoc] = useState<LocationValue>(emptyLocation)

  const onSubmit = async (ev: React.FormEvent) => {
    ev.preventDefault()
    try {
      const created = await create.mutateAsync({
        title_bn: titleBN.trim(),
        title_en: titleEN.trim() || null,
        summary_bn: summaryBN.trim() || null,
        summary_en: summaryEN.trim() || null,
        incident_date: incidentDate || null,
        country_id: loc.countryId,
        division_id: loc.divisionId,
        district_id: loc.districtId,
        upazila_id: loc.upazilaId,
        location_text: loc.text || null,
        crime_type_id: crimeTypeID ? Number(crimeTypeID) : null,
        tags: tags
          .split(',')
          .map((t) => t.trim())
          .filter(Boolean),
      } as Record<string, unknown>)
      nav(`/me/cases/${created.id}/edit`, { replace: true })
    } catch {
      // Surface error below.
    }
  }

  const errMsg = create.error
    ? create.error instanceof ApiError
      ? create.error.message
      : t('my_cases.error_generic')
    : null

  return (
    <Container width="reading" className="py-10">
      <header className="mb-6">
        <h1 className="font-display text-2xl font-semibold text-ink-900">
          {t('my_cases.new')}
        </h1>
        <p className="text-sm text-ink-600">{t('my_cases.new_subtitle')}</p>
      </header>

      <form onSubmit={onSubmit} noValidate>
        <Card>
          <CardHeader>
            <h2 className="text-sm font-medium text-ink-800">{t('my_cases.section_basics')}</h2>
          </CardHeader>
          <CardBody className="space-y-4">
            {errMsg && (
              <div role="alert" className="rounded-md bg-red-50 p-3 text-sm text-red-700">
                {errMsg}
              </div>
            )}
            <TextField
              label={t('my_cases.title_bn')}
              showRequired
              value={titleBN}
              onChange={(e) => setTitleBN(e.target.value)}
            />
            <TextField
              label={t('my_cases.title_en')}
              value={titleEN}
              onChange={(e) => setTitleEN(e.target.value)}
            />
            <TextArea
              label={t('my_cases.summary_bn')}
              rows={3}
              value={summaryBN}
              onChange={(e) => setSummaryBN(e.target.value)}
            />
            <TextArea
              label={t('my_cases.summary_en')}
              rows={3}
              value={summaryEN}
              onChange={(e) => setSummaryEN(e.target.value)}
            />
            <div className="grid gap-4 sm:grid-cols-2">
              <TextField
                label={t('my_cases.incident_date')}
                type="date"
                value={incidentDate}
                onChange={(e) => setIncidentDate(e.target.value)}
              />
              <Select
                label={t('my_cases.crime_type')}
                placeholder={t('my_cases.select_crime_type') ?? ''}
                options={(crimes.data ?? []).map((c) => ({
                  value: c.id,
                  label: isBN ? c.name_bn : c.name_en,
                }))}
                value={crimeTypeID}
                onChange={(e) => setCrimeTypeID(e.target.value)}
              />
            </div>
            <TextField
              label={t('my_cases.tags')}
              helperText={t('my_cases.tags_help')}
              placeholder="protest, harassment"
              value={tags}
              onChange={(e) => setTags(e.target.value)}
            />
          </CardBody>
        </Card>

        <Card className="mt-6">
          <CardHeader>
            <h2 className="text-sm font-medium text-ink-800">{t('my_cases.section_location')}</h2>
          </CardHeader>
          <CardBody>
            <LocationCascade value={loc} onChange={setLoc} />
          </CardBody>
        </Card>

        <div className="mt-6 flex justify-end gap-3">
          <Button type="button" variant="secondary" onClick={() => nav(-1)}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" loading={create.isPending}>
            {t('my_cases.save_draft')}
          </Button>
        </div>
      </form>
    </Container>
  )
}
