import type { ReactNode } from 'react'
import { Box, Stack, Typography } from '@mui/material'

interface PageHeaderProps {
  eyebrow: string
  title: string
  summary: string
  actions?: ReactNode
}

export function PageHeader({ eyebrow, title, summary, actions }: PageHeaderProps) {
  return (
    <Box component="header" className="page-header">
      <Stack direction={{ xs: 'column', md: 'row' }} justifyContent="space-between" alignItems={{ xs: 'stretch', md: 'flex-end' }} gap={2}>
        <Box>
          <Typography className="eyebrow">{eyebrow}</Typography>
          <Typography component="h1" variant="h4">{title}</Typography>
          <Typography className="page-summary">{summary}</Typography>
        </Box>
        {actions && <Stack direction="row" gap={1} flexWrap="wrap">{actions}</Stack>}
      </Stack>
    </Box>
  )
}

