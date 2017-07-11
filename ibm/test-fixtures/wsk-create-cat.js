function main(params) {

    return new Promise(function(resolve, reject) {

        if (!params.name) {
            reject({
                'error': 'name parameter not set.'
            });
        } else {
            resolve({
                id: 1
            });
        }

    });

}