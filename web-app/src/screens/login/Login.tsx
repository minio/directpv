import {Button, LoginWrapper} from 'mds';
import React, {Fragment} from 'react';

const demoInputStyles = {
    width: "100%",
    backgroundColor: "transparent",
    border: "#00000020 1px solid",
    borderRadius: "3px",
    height: "30px",
    marginBottom: "20px",
    padding: "5px",
};

const Login = () => {
    return (<LoginWrapper
            promoHeader={<Fragment>Multi-Cloud Object&nbsp;Store</Fragment>}
            promoInfo={
                <Fragment>
                    MinIO offers high-performance, S3 compatible object storage. <br/>
                    Native to Kubernetes, MinIO is the only object storage suite available on
                    every public cloud, every Kubernetes distribution, the private cloud and
                    the edge. MinIO is software-defined and is 100% open source under GNU AGPL
                    v3. <a href={"#"}>link</a>
                </Fragment>
            }
            logoProps={{
                applicationName: "console",
                subVariant: "AGPL",
            }}
            form={
                <Fragment>
                    DEMO FORM
                    <input name={"testInput"} style={demoInputStyles} placeholder="User"/>
                    <br/>
                    <input
                        name={"testInput"}
                        type={"password"}
                        style={demoInputStyles}
                        placeholder="Password"
                    />
                    <br/>
                    <Button
                        id={"submit"}
                        type={"button"}
                        label={"Login"}
                        variant={"callAction"}
                        fullWidth
                    />
                </Fragment>
            }
            formFooter={
                <Fragment>
                    Documentation│<a href={"#"}>GitHub</a>│Support│Download
                </Fragment>
            }
        />
    )
}

export default Login;