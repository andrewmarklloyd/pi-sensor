import {
    Box,
    Button,
    Container,
    Grid,
    Stack
} from "@mui/material";
import CodeIcon from '@mui/icons-material/Code';

export const Footer = () => {
  return (
    <Box
      sx={{
        width: "100%",
        height: "auto",
        paddingTop: "1rem",
        paddingBottom: "1rem",
      }}
    >
      <Container>
        <Grid container direction="column" alignItems="center">
          <Stack direction="row" spacing={3}>
            <Button href={"https://github.com/andrewmarklloyd/pi-sensor/commit/"+process.env.PUBLIC_REACT_APP_VERSION} variant="outlined">
                <CodeIcon></CodeIcon> App Version {process.env.PUBLIC_REACT_APP_VERSION}
            </Button>
            <img src="https://github.com/andrewmarklloyd/pi-sensor/actions/workflows/main.yml/badge.svg" alt="build-badge"></img>
          </Stack>
        </Grid>
      </Container>
    </Box>
  );
};

export default Footer;