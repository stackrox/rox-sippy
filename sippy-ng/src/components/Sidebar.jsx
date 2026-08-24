import {
  AppsOutage,
  BugReport,
  ChevronRight,
  GitHub,
} from '@mui/icons-material'
import { DEFAULT_TEST_FILTERS } from '../constants'
import { LaunderedListItem } from './Laundry'
import { Link, useLocation } from 'react-router-dom'
import { ListItemButton, ListSubheader, useTheme } from '@mui/material'
import {
  pathForJobsWithFilter,
  pathForTestsWithFilter,
  safeEncodeURIComponent,
  withoutUnstable,
  withSort,
} from '../helpers'
import { SippyCapabilitiesContext } from '../App'
import { styled } from '@mui/styles'
import Collapse from '@mui/material/Collapse'
import Divider from '@mui/material/Divider'
import HomeIcon from '@mui/icons-material/Home'
import InfoIcon from '@mui/icons-material/Info'
import List from '@mui/material/List'
import ListIcon from '@mui/icons-material/List'
import ListItem from '@mui/material/ListItem'
import ListItemIcon from '@mui/material/ListItemIcon'
import ListItemText from '@mui/material/ListItemText'
import PropTypes from 'prop-types'
import React, { Fragment, useEffect, useMemo, useRef } from 'react'
import SearchIcon from '@mui/icons-material/Search'
import SippyLogo from './SippyLogo'

const StyledListItemButton = styled(ListItemButton)({
  padding: 0,
  margin: 0,
})

function groupReleasesByProduct(releases, releaseAttrs) {
  const groups = {}
  releases.forEach((release) => {
    const product = releaseAttrs?.[release]?.product || 'Other'
    if (!groups[product]) {
      groups[product] = []
    }
    groups[product].push(release)
  })

  const sortedProducts = Object.keys(groups).sort((a, b) => {
    if (a === 'ACS') return -1
    if (b === 'ACS') return 1
    return a.localeCompare(b)
  })

  return sortedProducts.map((product) => ({
    product,
    releases: groups[product],
  }))
}

export default function Sidebar(props) {
  const classes = useTheme()
  const location = useLocation()

  const [openReleases, setOpenReleases] = React.useState({})
  const [openGroups, setOpenGroups] = React.useState({})
  const initialPathRef = useRef(location.pathname)

  const productGroups = useMemo(
    () =>
      groupReleasesByProduct(
        props.releaseConfig.releases || [],
        props.releaseConfig.release_attrs
      ),
    [props.releaseConfig]
  )

  useEffect(() => {
    const parts = initialPathRef.current.split('/')
    const tmpOpenReleases = {}
    const tmpOpenGroups = {}

    productGroups.forEach(({ product }) => {
      tmpOpenGroups[product] = product === 'ACS'
    })

    if (parts.length >= 3) {
      const release = parts[2]
      const group = productGroups.find((g) => g.releases.includes(release))
      if (group) {
        tmpOpenGroups[group.product] = true
        tmpOpenReleases[releaseKey(group.product, release)] = true
      }
    } else if (props.defaultRelease) {
      const group = productGroups.find((g) =>
        g.releases.includes(props.defaultRelease)
      )
      if (group) {
        tmpOpenGroups[group.product] = true
        tmpOpenReleases[releaseKey(group.product, props.defaultRelease)] = true
      }
    } else if (
      productGroups.length > 0 &&
      productGroups[0].releases.length > 0
    ) {
      tmpOpenGroups[productGroups[0].product] = true
      tmpOpenReleases[
        releaseKey(productGroups[0].product, productGroups[0].releases[0])
      ] = true
    }

    setOpenGroups(tmpOpenGroups)
    setOpenReleases(tmpOpenReleases)
  }, [productGroups, props.defaultRelease])

  function handleGroupClick(product) {
    setOpenGroups((prev) => ({ ...prev, [product]: !prev[product] }))
  }

  function releaseKey(product, release) {
    return `${product}::${release}`
  }

  function handleReleaseClick(product, release) {
    const key = releaseKey(product, release)
    setOpenReleases((prev) => ({ ...prev, [key]: !prev[key] }))
  }

  function reportAnIssueURI() {
    const description = `Describe your feature request or bug:\n\n

    Relevant Sippy URL:\n
    ${window.location.href}\n\n`
    return `https://redhat.atlassian.net/secure/CreateIssueDetails!init.jspa?pid=11604&issuetype=10009&description=${safeEncodeURIComponent(
      description
    )}`
  }

  function renderReleaseItems(release) {
    return (
      <List component="div" disablePadding sx={{ pl: 3 }}>
        <ListItem
          key={'release-overview-' + release}
          component={Link}
          to={'/release/' + release}
          className={classes.nested}
        >
          <StyledListItemButton>
            <ListItemIcon>
              <InfoIcon />
            </ListItemIcon>
            <ListItemText primary="Overview" />
          </StyledListItemButton>
        </ListItem>
        <ListItem
          key={'release-jobs-' + release}
          component={Link}
          to={withSort(
            pathForJobsWithFilter(release, {
              items: [...withoutUnstable()],
            }),
            'net_improvement',
            'asc'
          )}
          className={classes.nested}
        >
          <StyledListItemButton>
            <ListItemIcon>
              <ListIcon />
            </ListItemIcon>
            <ListItemText primary="Jobs" />
          </StyledListItemButton>
        </ListItem>

        {props.releaseConfig.release_attrs?.[release]?.capabilities
          ?.pullRequests && (
          <ListItem
            key={'release-pull-requests-' + release}
            component={Link}
            to={`/pull_requests/${release}`}
            className={classes.nested}
          >
            <StyledListItemButton>
              <ListItemIcon>
                <GitHub />
              </ListItemIcon>
              <ListItemText primary="Pull Requests" />
            </StyledListItemButton>
          </ListItem>
        )}

        <ListItem
          key={'release-tests-' + release}
          component={Link}
          to={withSort(
            pathForTestsWithFilter(release, {
              items: DEFAULT_TEST_FILTERS,
              linkOperator: 'and',
            }),
            'net_improvement',
            'asc'
          )}
          className={classes.nested}
        >
          <StyledListItemButton>
            <ListItemIcon>
              <SearchIcon />
            </ListItemIcon>
            <ListItemText primary="Tests" />
          </StyledListItemButton>
        </ListItem>
      </List>
    )
  }

  return (
    <Fragment>
      <List>
        <ListItem component={Link} to="/" key="Home">
          <StyledListItemButton>
            <ListItemIcon>
              <HomeIcon />
            </ListItemIcon>
            <ListItemText primary="Home" />
          </StyledListItemButton>
        </ListItem>
      </List>
      <SippyCapabilitiesContext.Consumer>
        {(value) => {
          if (value.includes('local_db')) {
            return (
              <Fragment>
                <Divider />
                <List
                  subheader={
                    <ListSubheader component="div" id="Overall Components">
                      Tools
                    </ListSubheader>
                  }
                >
                  <ListItem
                    key={'release-health-'}
                    component={Link}
                    to={'/component_readiness/main'}
                    className={classes.nested}
                  >
                    <StyledListItemButton>
                      <ListItemIcon>
                        <AppsOutage />
                      </ListItemIcon>
                      <ListItemText primary="Component Readiness" />
                    </StyledListItemButton>
                  </ListItem>
                </List>
              </Fragment>
            )
          }
        }}
      </SippyCapabilitiesContext.Consumer>

      <SippyCapabilitiesContext.Consumer>
        {(value) => {
          if (value.includes('local_db')) {
            return (
              <Fragment>
                <Divider />
                <List
                  subheader={
                    <ListSubheader component="div" id="releases">
                      Releases
                    </ListSubheader>
                  }
                >
                  {productGroups.map(({ product, releases }) => (
                    <Fragment key={'product-group-' + product}>
                      <ListItem onClick={() => handleGroupClick(product)}>
                        <StyledListItemButton>
                          <ChevronRight
                            sx={{
                              transition: 'transform 0.2s',
                              transform: openGroups[product]
                                ? 'rotate(90deg)'
                                : 'none',
                            }}
                          />
                          <ListItemText
                            primary={product}
                            primaryTypographyProps={{ fontWeight: 'bold' }}
                          />
                        </StyledListItemButton>
                      </ListItem>
                      <Collapse
                        in={openGroups[product]}
                        timeout="auto"
                        unmountOnExit
                      >
                        <List component="div" disablePadding>
                          {releases.map((release) => (
                            <Fragment key={'section-release-' + release}>
                              <ListItem
                                onClick={() =>
                                  handleReleaseClick(product, release)
                                }
                                sx={{ pl: 3 }}
                              >
                                <StyledListItemButton>
                                  <ChevronRight
                                    fontSize="small"
                                    sx={{
                                      transition: 'transform 0.2s',
                                      transform: openReleases[
                                        releaseKey(product, release)
                                      ]
                                        ? 'rotate(90deg)'
                                        : 'none',
                                    }}
                                  />
                                  <ListItemText primary={release} />
                                </StyledListItemButton>
                              </ListItem>
                              <Collapse
                                in={openReleases[releaseKey(product, release)]}
                                timeout="auto"
                                unmountOnExit
                              >
                                {renderReleaseItems(release)}
                              </Collapse>
                            </Fragment>
                          ))}
                        </List>
                      </Collapse>
                    </Fragment>
                  ))}
                </List>
              </Fragment>
            )
          }
        }}
      </SippyCapabilitiesContext.Consumer>

      <Divider />
      <List
        subheader={
          <ListSubheader component="div" id="resources">
            Resources
          </ListSubheader>
        }
      >
        <LaunderedListItem
          component="a"
          target="_blank"
          address={reportAnIssueURI()}
          key="ReportAnIssue"
        >
          <ListItemIcon>
            <BugReport />
          </ListItemIcon>
          <ListItemText primary="Report an Issue" />
        </LaunderedListItem>

        <ListItem
          component="a"
          target="_blank"
          href="https://www.github.com/stackrox/acs-sippy"
          key="GitHub"
        >
          <ListItemIcon>
            <GitHub />
          </ListItemIcon>
          <ListItemText primary="GitHub Repo" />
        </ListItem>
        <Divider />
        <div align="center">
          <SippyLogo />
        </div>
      </List>
    </Fragment>
  )
}

Sidebar.propTypes = {
  releaseConfig: PropTypes.object,
  defaultRelease: PropTypes.string,
}
