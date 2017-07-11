function main(params) {

    return new Promise(function(resolve, reject) {

        if (!params.id) {
            reject({
                'error': 'id parameter not set.'
            });
        } else {
            resolve({
                id: params.id,
                name: 'Tahoma',
                color: 'Tabby'
            });
        }

    });

}