import {
  Alert,
  Button,
  CircularProgress,
  Snackbar,
  Tooltip,
} from '@mui/material'
import RefreshIcon from '@mui/icons-material/Refresh'
import React, { useState } from 'react'

export default function RefreshButton() {
  const [loading, setLoading] = useState(false)
  const [snackbar, setSnackbar] = useState({
    open: false,
    message: '',
    severity: 'info',
  })

  const handleRefresh = async () => {
    setLoading(true)
    try {
      const response = await fetch(
        import.meta.env.VITE_API_URL + '/api/refresh',
        {
          method: 'GET',
        }
      )

      const data = await response.json()

      if (response.ok) {
        setSnackbar({
          open: true,
          message: data.message || 'Data refresh started successfully',
          severity: 'success',
        })
      } else if (response.status === 429) {
        setSnackbar({
          open: true,
          message:
            data.message || 'Refresh is rate-limited. Please try again later.',
          severity: 'warning',
        })
      } else {
        setSnackbar({
          open: true,
          message: data.message || 'Failed to trigger refresh',
          severity: 'error',
        })
      }
    } catch (error) {
      setSnackbar({
        open: true,
        message: 'Failed to connect to server: ' + error.message,
        severity: 'error',
      })
    } finally {
      setLoading(false)
    }
  }

  const handleCloseSnackbar = () => {
    setSnackbar({ ...snackbar, open: false })
  }

  return (
    <>
      <Tooltip title="Trigger on-demand data sync from BigQuery">
        <Button
          variant="outlined"
          size="small"
          startIcon={loading ? <CircularProgress size={16} /> : <RefreshIcon />}
          onClick={handleRefresh}
          disabled={loading}
        >
          Refresh Data
        </Button>
      </Tooltip>
      <Snackbar
        open={snackbar.open}
        autoHideDuration={6000}
        onClose={handleCloseSnackbar}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
      >
        <Alert
          onClose={handleCloseSnackbar}
          severity={snackbar.severity}
          sx={{ width: '100%' }}
        >
          {snackbar.message}
        </Alert>
      </Snackbar>
    </>
  )
}
